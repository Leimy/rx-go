package main

import (
	"fmt"
	"io"

	"log"
	"net/http"
	"sync"

	"github.com/Leimy/rx-go/bot"
	"github.com/Leimy/rx-go/nowplaying"
	"github.com/Leimy/rx-go/twit"
)

var nowPlaying *nowplaying.NowPlaying
var botFrom chan string
var botTo chan string
var tweeter twit.Tweeter

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

type autoState struct {
	sync.RWMutex
	tweet bool
	last  bool
}

func newAutoState() *autoState {
	return &autoState{sync.RWMutex{}, false, false}
}

func (as *autoState) getTweet() bool {
	as.RLock()
	defer as.RUnlock()
	return as.tweet
}

func (as *autoState) getLast() bool {
	as.RLock()
	defer as.RUnlock()
	return as.last
}

func (as *autoState) toggleTweet() {
	as.Lock()
	defer as.Unlock()
	as.tweet = !as.tweet
}

func (as *autoState) toggleLast() {
	as.Lock()
	defer as.Unlock()
	as.last = !as.last
}

var autos *autoState

func metaSubscriber() {
	updates := make(chan string)
	nowPlaying.Subscribe(updates)
	for {
		for line := range updates {
			if autos.getTweet() {
				tweeter(line)
			}
			if autos.getLast() {
				botTo <- line
			}
		}
	}
}

func procLine(line string) {
	log.Printf("got: %q", line)
	switch line {
	case "?lastsong?":
		botTo <- nowPlaying.Get()
	case "?tweet?":
		tweeter(nowPlaying.Get())
	case "?autotweet?":
		autos.toggleTweet()
		botTo <- fmt.Sprintf("Autotweet is %v", autos.getTweet())
	case "?autolast?":
		autos.toggleLast()
		botTo <- fmt.Sprintf("Autolast is %v", autos.getLast())
	}
}

// Keeps the bot alive, never returns
func keepBotAlive() {
	botFrom = make(chan string)
	botTo = make(chan string)
	start := func() {
		defer close(botTo)
		bot.NewBot("#radioxenu", "son_of_metabot", "irc.radioxenu.com:6667", botFrom, botTo)
	}
	for {
		go start()
		for line := range botFrom {
			procLine(line)
		}
	}
}

func init() {
	nowPlaying = nowplaying.NewNowPlaying("http://radioxenu.com:8000/relay")
	tweeter = twit.MakeTweeter("@radioxenu http://tunein.com/radio/Radio-Xenu-s118981/")
	autos = newAutoState()
	handleTwit("/tweet")
	handleMeta()
}

func main() {
	go nowPlaying.StartUpdating()
	go metaSubscriber()
	go keepBotAlive()
	log.Fatal(http.ListenAndServe(":8080", nil))
}
