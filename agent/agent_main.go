package main

import (
	"bytes"
	"clearview/agent/data"
	"clearview/common"
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/tklauser/go-sysconf"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	DEFAULT_SLEEP    = 60
	DEFAULT_PORT     = 443
	DEFAULT_INSECURE = false

	CONTENT_TYPE          = "Content-Type"
	CONTENT_ENCODING      = "Content-Encoding"
	CONTENT_ENCODING_GZIP = "gzip"

	REQ_MIME_APPLICATION_JSON = "application/json"

	DEFAULT_CONFIG_FILE = "/etc/clearview.conf"
)

type Response struct {
	Die   string `json:"die"`
	Sleep int    `json:"sleep"`
}

type Flags struct {
	account        string
	server         string
	port           int
	sleep          int
	insecure       bool
	configFileName string
}

func (f *Flags) makeServerUrl(confServerAddr string) url.URL {
	u := url.URL{}
	if f.insecure {
		u.Scheme = "http"
	} else {
		u.Scheme = "https"
	}
	server := confServerAddr
	if f.server != "" {
		server = f.server
	}
	u.Host = server

	if u.Port() == "" {
		if !f.insecure && f.port != 443 {
			u.Host += ":" + strconv.Itoa(f.port)
		}
	}

	return u
}

func main() {
	flags := Flags{}

	flag.StringVar(&flags.server, "s", "", "Server address")
	flag.IntVar(&flags.port, "p", DEFAULT_PORT, "Server port")
	flag.IntVar(&flags.sleep, "d", DEFAULT_SLEEP, "Duration of sleep between data points")
	flag.BoolVar(&flags.insecure, "i", DEFAULT_INSECURE, "Do not use tls (https) for connecting to server")
	flag.StringVar(&flags.configFileName, "f", DEFAULT_CONFIG_FILE, "The config file to use")
	flag.Parse()

	//nargs := flag.NArg()
	//args := flag.Args()
	fmt.Printf("Reading config from %s\n", flags.configFileName)

	config := common.ReadDefaultConfigFile(flags.configFileName)
	if err := config.GetError(); err != nil {
		fmt.Printf("Cannot read config file %s: %v\n", flags.configFileName, err)
		os.Exit(1)
	}

	configServerAddr := config.GetOrDefault("server_addr", "")
	flags.insecure = flags.insecure || config.GetBoolean("insecure", false)

	serverUrl := flags.makeServerUrl(configServerAddr)
	fmt.Printf("Server url: %s\n", serverUrl.String())

	// HTTP client
	client := &http.Client{Timeout: 5 * time.Second}

	// Default sleep time
	sleep := flags.sleep

	// We calculate some items as diff between curr and prev, e.g. cpu usage
	var dataPrev *common.Data = nil

	for {
		ticks := uint64(0)
		uptime := uint64(0)

		// Get ticks, used by several types of stats
		ticksValue, err := sysconf.Sysconf(sysconf.SC_CLK_TCK)
		if ticksValue <= 0 || err != nil {
			fmt.Printf("Cannot get ticks value: %v, assuming 100Hz\n", err)
			ticksValue = 100
		} else {
			ticks = uint64(ticksValue)
		}

		// Get uptime
		if procUptime, err := ioutil.ReadFile("/proc/uptime"); err == nil {
			l := strings.Fields(string(procUptime))
			if len(l) > 0 {
				if t, err := strconv.ParseFloat(l[0], 64); err == nil {
					uptime = uint64(t)
				}
			}
		}

		// Data item
		now := time.Now()
		dataCurr := &common.Data{TicksPerSecond: ticks, UptimeTicks: uptime, WhenTime: now, WhenSeconds: now.Unix()}

		// Fill it in
		_ = data.GetDataMemory(dataCurr)
		_ = data.GetDataCPU(dataCurr)

		_ = data.GetDataNetwork(dataCurr)
		_ = data.GetDataDisks(dataCurr)

		_ = data.GetDataProcessesPorts(dataCurr)

		_ = data.GetDataSysInfo(dataCurr)
		_ = data.GetDataAppNginx(client, &config, dataCurr)
		_ = data.GetDataAppApache(client, &config, dataCurr)
		_ = data.GetDataAppMysql(client, &config, dataCurr)
		_ = data.GetDataAppPgsql(client, &config, dataCurr)

		// Compute diff
		if dataPrev != nil {
			dataCurr.CalculateInstant(dataPrev)
			dataCurr.FilterProcessList()

			durationMillis := uint64(0)
			if dataCurr.WhenTime.After(dataPrev.WhenTime) {
				diff := dataCurr.WhenTime.Sub(dataPrev.WhenTime)
				durationMillis = (uint64(diff) + uint64(time.Millisecond/2)) / uint64(time.Millisecond)
			}
			dataCurr.DurationMillis = durationMillis
		}

		// Send to server
		sleepNew, die, err := sendDataToServer(client, serverUrl, &config, dataCurr)
		if err != nil {
			fmt.Printf("Error sending data: %s\n", err)
			sleep = flags.sleep
		} else {
			fmt.Printf("Sent data to %s\n", serverUrl.String())
			if sleepNew > 0 {
				sleep = sleepNew
			}
		}

		// Current becomes previous
		dataPrev = dataCurr

		// Server tells us to quit
		if die {
			fmt.Printf("Server told us to quit\n")
			break
		}

		// Wait / sleep
		fmt.Printf("Sleeping for %d seconds\n", sleep)
		time.Sleep(time.Duration(sleep) * time.Second)
	}
}

var gzipWriterPool = sync.Pool{
	New: func() interface{} {
		return gzip.NewWriter(ioutil.Discard)
	},
}

func sendDataToServer(client *http.Client, url url.URL, config *common.Config, data *common.Data) (int, bool, error) {
	confNodeId := config.Get("node_id")
	if confNodeId == "" {
		return 0, false, fmt.Errorf("need node_id in config file")
	}

	post := common.PostData{
		Version: 1,
		Node:    confNodeId,
		Payload: data,
	}

	sendBytesPayload, err := json.Marshal(&post)
	if err != nil {
		return 0, false, err
	}

	sendBytesBuffer := bytes.Buffer{}
	sendBytesWriter := io.Writer(&sendBytesBuffer)

	// Use a pooled gzip writer
	compressWriter := gzipWriterPool.Get().(*gzip.Writer)
	defer gzipWriterPool.Put(compressWriter)

	compressWriter.Reset(sendBytesWriter)

	_, _ = compressWriter.Write(sendBytesPayload)
	_ = compressWriter.Flush()
	_ = compressWriter.Close()

	sendBytesSlice := sendBytesBuffer.Bytes()
	body := bytes.NewReader(sendBytesSlice)

	url.Path = path.Join(url.Path, "/cv/agent/v1/add")

	req, err := http.NewRequest("POST", url.String(), body)
	if err != nil {
		return 0, false, err
	}

	req.Header.Set(CONTENT_TYPE, REQ_MIME_APPLICATION_JSON)
	req.Header.Set(CONTENT_ENCODING, CONTENT_ENCODING_GZIP)

	authUserName := config.Get(common.CONFIG_AUTH_USERNAME)
	authPassword := config.Get(common.CONFIG_AUTH_PASSWORD)
	if authUserName != "" && authPassword != "" {
		req.SetBasicAuth(authUserName, authPassword)
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Cannot get http response data: %s\n", err)
		return 0, false, err
	}

	defer func() {
		if resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Status code: %d, %s\n", resp.StatusCode, resp.Status)
		return 0, false, fmt.Errorf("status code: %d", resp.StatusCode)
	}

	// Parse response JSON
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Cannot get http response data: %s\n", err)
		return 0, false, err
	}

	sleepNew := 0
	die := false

	if len(respBody) > 0 {
		var resp Response
		if json.Unmarshal(respBody, &resp) == nil {
			die = resp.Die == "please"
			sleepNew = resp.Sleep
		}
	}

	return sleepNew, die, nil
}
