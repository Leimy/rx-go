package nowplaying

import (
	"github.com/Leimy/rx-go/meta"
	"log"
	"sync"
	"time"
)

// NowPlaying represent the song now playing on a Icy-MetaData compatible
// SHOUT stream
type NowPlaying struct {
	sync.RWMutex
	song string
}

// Get returns the currently playing song
func (np *NowPlaying) Get() string {
	np.RLock()
	defer np.RUnlock()
	return np.song
}

func (np *NowPlaying) set(str string) {
	np.Lock()
	defer np.Unlock()
	np.song = str
}

// Owns "writing" end of the song data, and can auto-restart
func (np *NowPlaying) startUpdating(url string) {
	for {
		ch, _ := meta.StreamMeta(url)
		for metadata := range ch {
			log.Printf("Got new metadata: %s", metadata)
			np.set(metadata)
		}
		log.Printf("Metdata stream closed")
		time.Sleep(2 * time.Second)
	}
}

// NewNowPlaying creates a NowPlaying string taking an URL for a stream
func NewNowPlaying(url string) *NowPlaying {
	np := &NowPlaying{sync.RWMutex{}, ""}
	go np.startUpdating(url)
	return np
}
