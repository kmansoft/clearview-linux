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

const (
	WORKER_KEYS = "_SRWKDCLGI"
)

func GetDataAppApache(client *http.Client, config *common.Config, data *common.Data) error {

	if !data.HasProcess("apache2") && !data.HasProcess("httpd") && !data.HasProcess("httpd.bin") {
		return nil
	}

	data.AppApache = &common.DataAppApache{}

	location := config.GetOrDefault("apache_location", "http://127.0.0.1/server-status?auto")

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
	if strings.Index(respString, "Scoreboard:") < 0 {
		fmt.Printf("Strange looking apache response")
		return nil
	}

	exprDataLine := regexp.MustCompile(`([^:]+):\s+(.+)`)

	version := resp.Header.Get("Server")
	scoreBoard := ""

	for _, l := range strings.Split(respString, "\n") {
		if m := exprDataLine.FindStringSubmatch(string(l)); len(m) == 3 {
			key := m[1]
			value := m[2]
			if key == "Scoreboard" {
				scoreBoard = value
			} else if key == "ServerVersion" {
				if len(value) > 0 {
					version = value
				}
			} else if key == "Total Accesses" {
				data.AppApache.TotalAccessCountCumulative, _ = strconv.ParseUint(value, 10, 64)
			} else if key == "Total kBytes" {
				data.AppApache.TotalByteCountCumulative, _ = strconv.ParseUint(value, 10, 64)
				data.AppApache.TotalByteCountCumulative *= 1024
			}
		}
	}

	// Workers
	for _, ch := range scoreBoard {
		switch ch {
		case '_':
			data.AppApache.WorkersWaiting += 1
		case 'R':
			data.AppApache.WorkersReading += 1
		case 'W':
			data.AppApache.WorkersWriting += 1

		}
	}

	// Server version
	data.AppApache.Version = version

	return nil
}

func getWorkerKeyFromChar(ch int32) string {
	switch ch {
	case '_':
		return "Waiting for Connection"
	case 'S':
		return "Starting up"
	case 'R':
		return "Reading Request"
	case 'W':
		return "Sending Reply"
	case 'K':
		return "Keepalive"
	case 'D':
		return "DNS Lookup"
	case 'C':
		return "Closing connection"
	case 'L':
		return "Logging"
	case 'G':
		return "Gracefully finishing"
	case 'I':
		return "Idle cleanup of worker"
	}
	return ""
}
