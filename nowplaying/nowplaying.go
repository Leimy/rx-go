package nowplaying

import (
	"log"
	"sync"
	"time"

	"github.com/Leimy/rx-go/meta"
)

// NowPlaying represent the song now playing on a Icy-MetaData compatible
// SHOUT stream
type NowPlaying struct {
	sync.RWMutex
	song        string
	url         string
	subscribers []chan string
}

// Get returns the currently playing song
func (np *NowPlaying) Get() string {
	np.RLock()
	defer np.RUnlock()
	return np.song
}

// Subscribe by providing a channel of strings to hear about changes on
func (np *NowPlaying) Subscribe(c chan string) {
	np.Lock()
	defer np.Unlock()
	np.subscribers = append(np.subscribers, c)
}

func (np *NowPlaying) set(str string) {
	np.Lock()
	defer np.Unlock()
	np.song = str
}

// StartUpdating Owns "writing" end of the song data, and can auto-restart
// Doesn't return!  Run in a goroutine.
func (np *NowPlaying) StartUpdating() {
	// We own it because it's our write end
	defer func() {
		np.Lock()
		for c := range np.subscribers {
			close(np.subscribers[c])
		}
	}()

	for {
		ch, _ := meta.StreamMeta(np.url)
		for metadata := range ch {
			log.Printf("Got new metadata: %s", metadata)
			np.set(metadata)

			// update subscribers
			for c := range np.subscribers {
				np.subscribers[c] <- metadata
			}
		}
		log.Printf("Metadata stream closed")
		time.Sleep(2 * time.Second)
	}
}

// NewNowPlaying creates a NowPlaying string taking an URL for a stream
func NewNowPlaying(url string) *NowPlaying {
	np := &NowPlaying{sync.RWMutex{}, "", url, make([]chan string, 0)}
	return np
}
