package main

import (
	//"bufio"
	"fmt"
	"github.com/Leimy/rx-go/meta"
	//"github.com/Leimy/rx-go/twit"
	"log"
	"net/http"
	"time"
)

// Channels to monitor for when the "service" is shut down
var twitchan chan byte
var metaalive chan byte

// The main service shuts down here
var shutdown chan byte

func startMeta() {
	metaalive = make(chan byte)
	metachan := make(chan string)
	ch, err := meta.StreamMeta("http://radioxenu.com:8000/relay")
	if err != nil {
		close(metaalive)
	}
	go func() {
		defer close(metaalive)
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
	}()
	http.HandleFunc("/metadata", func(w http.ResponseWriter, r *http.Request) {
		nowPlaying := <-metachan
		fmt.Fprintf(w, "Now Playing: %s", nowPlaying)
	})
}

func startTwit(uri string) {
	twitchan = make(chan byte)
	//tweeter := twit.MakeTweeter("@radioxenu http://tunein.com/radio/Radio-Xenu-s118981/")
	http.HandleFunc(uri, func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var message = make([]byte, r.ContentLength)
		if _, err := r.Body.Read(message); err != nil {
			log.Printf("Failed to read request: %v", err)
			close(twitchan)
		}
		//		tweeter(string(message))
	})
}

// Start up endpoints for the http service, and restart on error
func watcher() {
	//startTwit("/tweet")
	startMeta()
	for {
		select {
		//case <-twitchan:
		//	startTwit("/tweet")
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
