package meta

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func parseIcy(rdr *bufio.Reader, c byte) (string, error) {
	numbytes := int(c) * 16
	bytes := make([]byte, numbytes)
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
		bufrdr := bufio.NewReaderSize(rdr, skip)
		for {
			skipbytes := make([]byte, skip)

			_, err := io.ReadFull(bufrdr, skipbytes)
			if err != nil {
				close(ch)
				break
			}
			c, err := bufrdr.ReadByte()
			if err != nil {
				close(ch)
			}
			if c > 0 {
				meta, err := parseIcy(bufrdr, c)
				if err != nil {
					close(ch)
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
