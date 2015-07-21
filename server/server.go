package main

import (
	//"bufio"
	"fmt"

	"log"
	"net/http"
	"time"

	"github.com/Leimy/rx-go/meta"
	"github.com/Leimy/rx-go/twit"
)

// Channels to monitor for when the "service" is shut down
var metaalive chan byte

// The main service shuts down here
var shutdown chan byte

func startMeta() {
	metachan := make(chan string)
	ch, err := meta.StreamMeta("http://radioxenu.com:8000/relay")
	if err != nil {
		close(metaalive)
	}
	metadataExtractor := func() {
		defer close(metachan)
		metadata := "Unknown"
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
	go metadataExtractor()
	http.HandleFunc("/metadata", func(w http.ResponseWriter, r *http.Request) {
	again:
		select {
		case nowPlaying, ok := <-metachan:
			if !ok {
				metachan = make(chan string)
				go metadataExtractor()
				nowPlaying = ""
				goto again
			} else {
				fmt.Fprintf(w, "Now Playing: %s", nowPlaying)
			}

		}
	})
}

func startTwit(uri string) {
	tweeter := twit.MakeTweeter("@radioxenu http://tunein.com/radio/Radio-Xenu-s118981/")
	http.HandleFunc(uri, func(w http.ResponseWriter, r *http.Request) {
		switch method := r.Method; method {
		case "PUT", "POST":
			defer r.Body.Close()
			var message = make([]byte, r.ContentLength)
			if _, err := r.Body.Read(message); err != nil {
				log.Printf("Failed to read request: %v", err)
			}
			tweeter(string(message))
		default:
			log.Printf("Unsupported method: %s", method)
		}
	})
}

// Start up endpoints for the http service, and restart on error
func watcher() {
	startTwit("/tweet")
	startMeta()
	for {
		select {
		case <-metaalive:
			go func() {
				time.Sleep(2000 * time.Millisecond)
				startMeta()
			}()
		case <-shutdown:
			return
		}
	}
}

func main() {
	shutdown := make(chan byte)
	go watcher()
	log.Fatal(http.ListenAndServe(":8080", nil))
	close(shutdown)
}
