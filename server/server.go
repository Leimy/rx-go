package main

import (
	"fmt"
	"io"

	"log"
	"net/http"
	"sync"
	"time"

	"github.com/Leimy/rx-go/bot"
	"github.com/Leimy/rx-go/nowplaying"
	"github.com/Leimy/rx-go/twit"
)

var nowPlaying *nowplaying.NowPlaying
var tweeter twit.Tweeter

var botTo struct {
	sync.RWMutex
	C chan string
}

func resetBotTo() {
	botTo.Lock()
	defer botTo.Unlock()
	botTo.C = make(chan string)
}

func getChan() (chan string) {
	botTo.RLock()
	defer botTo.RUnlock()
	return botTo.C
}

// just some counters
var stats struct {
	sync.RWMutex
	botRestarts uint64
}

func incBotRestarts() {
	stats.Lock()
	defer stats.Unlock()
	stats.botRestarts++
}

func getBotRestarts() uint64 {
	stats.RLock()
	defer stats.RUnlock()
	return stats.botRestarts
}

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

// Call only once per uri
func handleTwit(uri string, tweeter twit.Tweeter) {
	var message = make([]byte, 160)
	http.HandleFunc(uri, func(w http.ResponseWriter, r *http.Request) {
		switch method := r.Method; method {
		case "PUT", "POST":
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

// stats
func handleStats() {
	http.HandleFunc("/botstats", func(w http.ResponseWriter, r *http.Request) {
		switch method := r.Method; method {
		case "GET":
			fmt.Fprintf(w, "Restart count: %v", getBotRestarts())
		default:
			log.Printf("Unsupported method: %s", method)
		}
	})
}

// TODO-Maybe: module for the automatic behaviors. (only this server cares)
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

// Subscribes to nowPlaying, and reads from the subscription
// Takes actions based on the settings of automatic behaviors
func metaSubscriber() {
	updates := make(chan string)
	nowPlaying.Subscribe(updates)
	for {
		for line := range updates {
			if autos.getTweet() {
				// TODO: Use the REST endpoint
				tweeter(line)
			}
			if autos.getLast() {
				getChan() <- line
			}
		}
	}
}

// Commands from IRC
func procIRCLine(line string) {
	log.Printf("got: %q", line)
	switch line {
	case "?lastsong?":
		getChan() <- nowPlaying.Get()
	case "?tweet?":
		//TODO: use the REST endpoint
		tweeter(nowPlaying.Get())
	case "?autotweet?":
		autos.toggleTweet()
		getChan() <- fmt.Sprintf("Autotweet is %v", autos.getTweet())
	case "?autolast?":
		autos.toggleLast()
		getChan() <- fmt.Sprintf("Autolast is %v", autos.getLast())
	}
}

// Keeps the bot alive, never returns
func keepBotAlive() {

	start := func(done chan struct{}, botFrom chan string) {
		defer close(done)
		resetBotTo()
		//		bot.NewBot("#radioxenu", "son_of_metabot2", "irc.radioxenu.com:6667", botFrom, getChan())
		bot.NewBot("#radioxenu", "son_of_metabot2", "localhost:6667", botFrom, getChan())
	}
	for {
		done := make(chan struct{})
		botFrom := make(chan string)
		go start(done, botFrom)
		for line := range botFrom {
			procIRCLine(line)
		}
		<-done // wait until previous bot is dead before making another
		incBotRestarts()
		time.Sleep(5 * time.Second)
	}
}

func init() {
	nowPlaying = nowplaying.NewNowPlaying("http://radioxenu.com:8000/relay")
	autos = newAutoState()
	tweeter = twit.MakeTweeter("@radioxenu http://tunein.com/radio/Radio-Xenu-s118981/")
	handleTwit("/tweet", tweeter)
	handleMeta()
	handleStats()
}

func main() {
	go nowPlaying.StartUpdating()
	go metaSubscriber()
	go keepBotAlive()
	log.Fatal(http.ListenAndServe(":8080", nil))
}
