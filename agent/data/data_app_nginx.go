package data

import (
	"clearview/common"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

func GetDataAppNginx(client *http.Client, config *common.Config, data *common.Data) error {

	if !data.HasProcess("nginx") && !data.HasProcess("nginx.bin") {
		return nil
	}

	data.AppNginx = &common.DataAppNginx{}

	location := config.GetOrDefault("nginx_location", "http://127.0.0.1/nginx_status")

	resp, err := client.Get(location)
	if err != nil {
		fmt.Printf("Cannot get http response data: %s\n", err)
		return err
	}

	defer func() {
		if resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	// Parse response
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Cannot get http response data: %s\n", err)
		return err
	}

	respString := string(respBody)
	if strings.Index(respString, "server accepts handled requests") < 0 {
		fmt.Printf("Strange looking nginx response")
		return nil
	}

	exprActiveConnections := regexp.MustCompile(`Active connections: (\d+)`)
	exprTotalConnections := regexp.MustCompile(`(\d+) (\d+) (\d+)`)
	exprReadingWritingWaiting := regexp.MustCompile(`Reading: (\d+) Writing: (\d+) Waiting: (\d+)`)

	for _, l := range strings.Split(respString, "\n") {
		if m := exprActiveConnections.FindStringSubmatch(l); len(m) > 0 {
			// Active connections
			data.AppNginx.ConnActiveInstant, _ = strconv.ParseUint(m[1], 10, 64)
		} else if m := exprTotalConnections.FindStringSubmatch(l); len(m) > 0 {
			// Total connections / requests
			data.AppNginx.ConnAcceptedCumulative, _ = strconv.ParseUint(m[1], 10, 64)
			data.AppNginx.ConnHandledCumulative, _ = strconv.ParseUint(m[2], 10, 64)
		} else if m := exprReadingWritingWaiting.FindStringSubmatch(l); len(m) > 0 {
			// Waiting / reading / writing
			data.AppNginx.WorkersWaiting, _ = strconv.ParseUint(m[3], 10, 64)
			data.AppNginx.WorkersReading, _ = strconv.ParseUint(m[1], 10, 64)
			data.AppNginx.WorkersWriting, _ = strconv.ParseUint(m[2], 10, 64)
		}
	}

	// Server version
	version := resp.Header.Get("Server")
	if len(version) > 0 {
		data.AppNginx.Version = version
	}

	return nil
}
