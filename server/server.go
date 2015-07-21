package main

import (
	"fmt"
	"io"
	"time"

	"log"
	"net/http"

	"github.com/Leimy/rx-go/meta"
	"github.com/Leimy/rx-go/twit"
)

// The main service shuts down here
var shutdown chan byte

// Start up the metadata ripping service.
// Make a metadata extractor/forwarder to send it to
// When metadata is requested, forward the last known
// song to it.  The handler function can detect when the
// extractor is not running, and can repair/restart it.
// Only call this one time.
func startMeta() {
	metadataExtractor := func(metachan chan<- string) {
		ch, _ := meta.StreamMeta("http://radioxenu.com:8000/relay")
		defer close(metachan)
		metadata := ""
		ok := true
		for {
			select {
			case metadata, ok = (<-ch):
				if ok {
					log.Printf("Got new metadata: %s", metadata)
				} else {
					log.Printf("Metadata stream closed")

					return
				}
			case metachan <- metadata:
				log.Printf("Sent requested metadata: %s", metadata)
			}
		}
	}

	metachan := make(chan string)
	go metadataExtractor(metachan)

	http.HandleFunc("/metadata", func(w http.ResponseWriter, r *http.Request) {
		again := true
		for again {
			select {
			case nowPlaying, ok := <-metachan:
				if !ok {
					log.Printf("metadataExtractor is not running.  Start it.")
					metachan = make(chan string)
					go metadataExtractor(metachan)
					nowPlaying = ""
				} else {
					if nowPlaying == "" {
						time.Sleep(2 * time.Second)
					} else {
						fmt.Fprintf(w, "Now Playing: %s", nowPlaying)
						again = false
					}
				}

			}
		}
	})
}

// Call only one time
func startTwit(uri string) {
	tweeter := twit.MakeTweeter("@radioxenu http://tunein.com/radio/Radio-Xenu-s118981/")
	var message = make([]byte, 160)
	http.HandleFunc(uri, func(w http.ResponseWriter, r *http.Request) {
		switch method := r.Method; method {
		case "PUT", "POST":
			defer r.Body.Close()
			if _, err := r.Body.Read(message); err != nil && err != io.EOF {
				log.Printf("Failed to read request: %s %v", message, err)
			} else {
				log.Printf("Requested to tweet: %s", message)
				tweeter(string(message))
			}
		default:
			log.Printf("Unsupported method: %s", method)
		}
	})
}

func init() {
	startTwit("/tweet")
	startMeta()
}

func main() {
	log.Fatal(http.ListenAndServe(":8080", nil))
}
