package meta

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

var tempBuffer []byte

func parseIcy(rdr *bufio.Reader, c byte) (string, error) {
	numbytes := int(c) * 16
	// Check that our reused buffer is sized ok.
	if len(tempBuffer) < numbytes {
		tempBuffer = make([]byte, numbytes)
	}

	// Create a slice of exactly the size we need, and work with it.
	bytes := tempBuffer[:numbytes]

	n, err := io.ReadFull(rdr, bytes)
	if err != nil {
		return "", err
	}
	if n != numbytes {
		return "", errors.New("didn't get enough data")
	}
	return strings.Split(strings.Split(string(bytes), "=")[1], ";")[0], nil
}

func extractMetadata(rdr io.Reader, skip int) <-chan string {
	ch := make(chan string)
	go func() {
		defer close(ch)
		bufrdr := bufio.NewReaderSize(rdr, skip)
		skipbytes := make([]byte, skip)
		for {
			_, err := io.ReadFull(bufrdr, skipbytes)
			if err != nil {
				return
			}
			c, err := bufrdr.ReadByte()
			if err != nil {
				return
			}
			if c > 0 {
				meta, err := parseIcy(bufrdr, c)
				if err != nil {
					return
				}
				ch <- meta
			}
		}
	}()
	return ch
}

// StreamMeta takes a url to stream frun and returns a channel of metadata
// strings or an error.
func StreamMeta(url string) (<-chan string, error) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {

		return nil, err
	}

	req.Header.Add("Icy-MetaData", "1")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	amount := 0
	if _, err = fmt.Sscan(resp.Header.Get("Icy-Metaint"), &amount); err != nil {
		return nil, err
	}

	return extractMetadata(resp.Body, amount), nil
}
