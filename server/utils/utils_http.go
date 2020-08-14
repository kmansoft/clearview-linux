package utils

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Compression pool

var gzipWriterPool = sync.Pool{
	New: func() interface{} {
		return gzip.NewWriter(ioutil.Discard)
	},
}

var gzipReaderPool = sync.Pool{
	New: func() interface{} {
		return new(gzip.Reader)
	},
}

const (
	CONTENT_TYPE     = "Content-Type"
	CONTENT_ENCODING = "Content-Encoding"
	ACCEPT_ENCODING  = "Accept-Encoding"
	ENCODING_GZIP    = "gzip"

	MAX_HTTP_REQUEST_SIZE = 256 * 1024

	REQ_MIME_APPLICATION_JSON        = "application/json"
	RESP_MIME_APPLICATION_JSON_UTF_8 = "application/json; charset=UTF-8"
)

func StartHandlingHttpRequest(w http.ResponseWriter, req *http.Request) time.Time {
	now0 := time.Now()

	w.Header().Set("Cache-Control", "max-age=0, no-cache, no-store")
	w.Header().Set("Pragma", "no-cache")

	return now0
}

func FinishHandlingHttpRequest(w http.ResponseWriter, req *http.Request, status int, responseData string,
	now0 time.Time, responseMime string) {
	if status == 0 {
		// Done, the handler already wrote the result
	} else if status == http.StatusOK {
		w.Header().Set(CONTENT_TYPE, responseMime)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(responseData))
	} else if len(responseData) > 0 {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(responseData + "\r\n"))
	} else {
		w.WriteHeader(status)
	}

	now1 := time.Now()
	fmt.Printf("%s: handled %s %s, %d code, %d bytes\n", now1.Format(time.Stamp),
		now1.Sub(now0).Round(time.Millisecond), req.URL.String(), status, len(responseData))
}

func WriteResponseWithCompression(w http.ResponseWriter, r *http.Request, rsBytes []byte) bool {
	fmt.Printf("Returning data: %d bytes\n", len(rsBytes))

	acceptedEncodingList := strings.FieldsFunc(r.Header.Get(ACCEPT_ENCODING), func(r rune) bool {
		return r == ',' || r == ' '
	})
	for _, acceptedEncoding := range acceptedEncodingList {
		if acceptedEncoding == ENCODING_GZIP {
			byteWriter := bytes.Buffer{}

			gzipWriter := gzipWriterPool.Get().(*gzip.Writer)
			gzipWriter.Reset(&byteWriter)

			_, _ = gzipWriter.Write(rsBytes)
			_ = gzipWriter.Flush()

			fmt.Printf("Compressed to %d bytes\n", byteWriter.Len())

			w.Header().Set(CONTENT_ENCODING, ENCODING_GZIP)
			w.WriteHeader(http.StatusOK)

			_, _ = w.Write(byteWriter.Bytes())

			gzipWriterPool.Put(gzipWriter)

			return true
		}
	}

	return false
}

func ReadRequestWithDecompression(compressed []byte) ([]byte, error) {
	gzipReader := gzipReaderPool.Get().(*gzip.Reader)
	defer gzipReaderPool.Put(gzipReader)

	err := gzipReader.Reset(bytes.NewReader(compressed))
	if err != nil {
		return nil, err
	}
	decompressed, err := ioutil.ReadAll(gzipReader)
	if err != nil {
		return nil, err
	}

	return decompressed, nil
}

func GetCookie(req *http.Request, name string) string {
	for _, c := range req.Cookies() {
		if c.Name == name {
			return c.Value
		}
	}
	return ""
}

func WriteJsonResponse(w http.ResponseWriter, rs interface{}) (int, string) {
	rsBytes, err := json.Marshal(rs)
	if err != nil {
		return http.StatusInternalServerError, err.Error()
	}

	fmt.Printf("Returning data:\n-----\n%s\n-----\n", rsBytes)

	return http.StatusOK, string(rsBytes)
}
