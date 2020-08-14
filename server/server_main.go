package main

import (
	"clearview/common"
	"clearview/server/defs"
	"clearview/server/service"
	"clearview/server/utils"
	"context"
	"crypto/subtle"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/julienschmidt/httprouter"
)

const (
	DEFAULT_PORT             = 63001
	DEFAULT_ADDR             = "127.0.0.1"
	DEFAULT_VERBOSE          = false
	DEFAULT_SLEEP            = 60
	DEFAULT_HTTP_ROOT_DIR    = ""
	DEFAULT_HTTP_CV_DIR      = "/var/lib/clearview/site/cv"
	DEFAULT_CONFIG_FILE_NAME = "/etc/clearview-server.conf"
)

func main() {
	flags := defs.Flags{}

	flag.StringVar(&flags.ListenAddr, "a", DEFAULT_ADDR, "Server listen address")
	flag.IntVar(&flags.ListenPort, "p", DEFAULT_PORT, "Server listen port")

	flag.BoolVar(&flags.Verbose, "v", DEFAULT_VERBOSE, "Verbose output")

	flag.IntVar(&flags.SleepInterval, "s", DEFAULT_SLEEP, "Sleep interval")

	flag.StringVar(&flags.HttpWebRootDir, "rootdir", DEFAULT_HTTP_ROOT_DIR, "Directory for / pages")
	flag.StringVar(&flags.HttpWebCvDir, "cvdir", DEFAULT_HTTP_CV_DIR, "Directory for /cv/web pages")

	flag.StringVar(&flags.InfluxDbUri, "influx-db-uri", service.DATABASE_URI_INFLUX, "Influx database uri")
	flag.StringVar(&flags.InfluxDbName, "influx-db-name", service.DATABASE_NAME_INFLUX, "Influx database name")

	flag.StringVar(&flags.MongoServerName, "mongo-server-name", service.DATABASE_SERVER_MONGO, "Mongo database server")
	flag.StringVar(&flags.MongoDbName, "mongo-db-name", service.DATABASE_NAME_MONGO, "Mongo database name")

	flag.StringVar(&flags.ConfigFileName, "f", DEFAULT_CONFIG_FILE_NAME, "Config file name")

	flag.BoolVar(&flags.DemoMode, "demo", false, "Demo mode")

	flag.Parse()

	// We'll need an http server
	addr := fmt.Sprintf("%s:%d", flags.ListenAddr, flags.ListenPort)
	fmt.Printf("Starting the server on %s\n", addr)

	router := httprouter.New()
	router.HandleOPTIONS = false

	serverObj := &http.Server{
		Addr:    addr,
		Handler: router,
	}
	serverLock := sync.Mutex{}

	// Load config
	config := common.ReadDefaultConfigFile(flags.ConfigFileName)
	if err := config.GetError(); err != nil {
		fmt.Printf("Cannot read config file %s: %v\n", flags.ConfigFileName, err)
		os.Exit(1)
	}

	authUserName := config.Get(common.CONFIG_AUTH_USERNAME)
	authPassword := config.Get(common.CONFIG_AUTH_PASSWORD)

	if !flags.DemoMode && (authUserName == "" || authPassword == "") {
		fmt.Printf("Please specify %s and %s in %s or enable demo mode\n",
			common.CONFIG_AUTH_USERNAME, common.CONFIG_AUTH_PASSWORD, flags.ConfigFileName)
		os.Exit(1)
	}

	// Create api service
	apiService := service.NewApiService(flags.DemoMode)

	// Signals
	signalChan := make(chan os.Signal)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	go func(c chan os.Signal) {
		var signalCount int32 = 1
		for s := range c {
			fmt.Printf("Signal: %s\n", s)
			if atomic.AddInt32(&signalCount, 1) >= 3 {
				fmt.Printf("Signal count is %d, exiting the process\n", signalCount)
				os.Exit(1)
			}

			serverLock.Lock()

			server := serverObj
			serverObj = nil

			serverLock.Unlock()

			if server != nil {
				fmt.Printf("Shutting down the server\n")

				ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)

				_ = server.Shutdown(ctx)

				cancel()
			}

		}
	}(signalChan)
	defer close(signalChan)

	// Models
	fmt.Printf("InfluxDB connection: %q, database: %q\n", flags.InfluxDbUri, flags.InfluxDbName)
	fmt.Printf("Mongo server: %q, database: %q\n", flags.MongoServerName, flags.MongoDbName)

	// API service
	apiModel, err := service.NewApiServerModel(flags)
	if err != nil {
		fmt.Printf("Cannot create api model: %s\n", err)
		os.Exit(1)
	}

	apiService.Setup(router, apiModel)

	myHandler := webHandler{
		authUserName: authUserName,
		authPassword: authPassword,
		demoMode:     flags.DemoMode,
		original:     serverObj.Handler,
		webRootDir:   path.Clean(flags.HttpWebRootDir),
		webCvDir:     path.Clean(flags.HttpWebCvDir),
	}
	serverObj.Handler = myHandler

	// Start http
	err = serverObj.ListenAndServe()

	if err != nil && err != http.ErrServerClosed {
		fmt.Printf("Cannot start http: %s\n", err)
		os.Exit(1)
	}
}

type webHandler struct {
	original     http.Handler
	webRootDir   string
	webCvDir     string
	authUserName string
	authPassword string
	demoMode     bool
}

func (h webHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	u := r.URL
	p := u.Path
	d := path.Dir(p)

	fmt.Println("Directory:", d)

	if !h.demoMode {
		if d != "/" {
			requestUserName, requestPassword, ok := r.BasicAuth()
			if !ok ||
				subtle.ConstantTimeCompare([]byte(requestUserName), []byte(h.authUserName)) != 1 ||
				subtle.ConstantTimeCompare([]byte(requestPassword), []byte(h.authPassword)) != 1 {
				// Not authenticated
				w.Header().Set("WWW-Authenticate", `Basic realm="ClearView"`)
				w.WriteHeader(http.StatusUnauthorized)
				time.Sleep(10 * time.Millisecond)
				return
			}
		}
	}

	if r.Method == "GET" {
		if d == "/cv/web" || strings.HasPrefix(d, "/cv/web/") {
			utils.StartHandlingHttpRequest(w, r)

			p = p[3:] // remove "/cv" prefix

			full := path.Join(h.webCvDir, p)
			http.ServeFile(w, r, full)
			return
		} else if d == "/" && h.webRootDir != "" {
			utils.StartHandlingHttpRequest(w, r)

			full := path.Join(h.webRootDir, p)
			http.ServeFile(w, r, full)
			return
		}
	}

	h.original.ServeHTTP(w, r)
}
