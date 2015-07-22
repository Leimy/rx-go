package main

import (
	"fmt"
	"io"

	"log"
	"net/http"

	"github.com/Leimy/rx-go/nowplaying"
	"github.com/Leimy/rx-go/twit"
)

var nowPlaying *nowplaying.NowPlaying

// Set up the metadata handler
func handleMeta() {
	http.HandleFunc("/metadata", func(w http.ResponseWriter, r *http.Request) {
		switch method := r.Method; method {
		case "GET":
			curSong := ""
			for curSong == "" {
				curSong = nowPlaying.Get()
			}
			fmt.Fprintf(w, "Now Playing: %s", curSong)
		default:
			log.Printf("%s Not implemented", method)
		}
	})
}

// Call only one time
func handleTwit(uri string) {
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
	nowPlaying = nowplaying.NewNowPlaying("http://radioxenu.com:8000/relay")
	handleTwit("/tweet")
	handleMeta()
}

func main() {
	log.Fatal(http.ListenAndServe(":8080", nil))
}
