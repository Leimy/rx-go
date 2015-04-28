// twitter service for 9p

package main

import (
	"flag"
	"github.com/Leimy/rx-go/twit"
	"github.com/mortdeus/go9p"
	"github.com/mortdeus/go9p/srv"
	"log"
	"os"
	"strings"
)

type Twitfs struct {
	srv   *srv.Fsrv
	user  go9p.User
	group go9p.Group
}

type TweetFile struct {
	srv.File
	data []byte
	tweeter twit.Tweeter
}

type TweetFileFactory struct {
	srv.File
	data []byte
}

func NewTweetFile(root *srv.File, name string, user go9p.User, group go9p.Group, mode uint32, template string) (error, *TweetFile) {
	twitterentry := new(TweetFile)
	twitterentry.tweeter = twit.MakeTweeter(template)
	err := twitterentry.Add(root, name, user, group, mode, twitterentry)
	if err != nil {
		return err, nil
	}
	return nil, twitterentry
}

const METADATA_SFX string = "on @radioxenu http://tunein.com/radio/Radio-Xenu-s118981/"

var addr = flag.String("a", "./crustysock", "unix domain socket path")
var debug = flag.Int("d", 0, "debuglevel")
var logsz = flag.Int("l", 2048, "log size")
var tsrv Twitfs
var root *srv.File

func init () {
	flag.Parse()
	tsrv.user = go9p.OsUsers.Uid2User(os.Geteuid())
	tsrv.group = go9p.OsUsers.Gid2Group(os.Getegid())
	root = new(srv.File)
	if err := root.Add(nil, "/", tsrv.user, nil, go9p.DMDIR|0555, nil); err != nil {
		log.Panic(err)
	}
}

// technically we should be reading the data into a buffer for this file
// and when it gets Clunked, send the message
func (t *TweetFile) Write(fid *srv.FFid, buf []byte, offset uint64) (int, error) {
	t.data = append(t.data, buf...)
	return len(buf), nil
}

func (t *TweetFile) Clunk(fid *srv.FFid) error {
	go func () {
		if err := t.tweeter(string(t.data)); err != nil {
			log.Printf("Error tweeting: %s\n", err)
		}
	}()
	log.Printf("Clunk: %p\n", fid)
	return nil
}

func (t *TweetFile) Remove(fid *srv.FFid) error {
	return nil
}


func (tff *TweetFileFactory) Write(fid *srv.FFid, buf []byte, offset uint64) (int, error) {
	tff.data = append(tff.data, buf...)
	return len(buf), nil
}

func (tff *TweetFileFactory) Clunk(fid *srv.FFid) error {
	s := string(tff.data)
	all := strings.SplitN(s, "|", 2)
	
	if len(all) != 2 {
		log.Printf("Illegal request, ignoring: %s\n", s)
	} else {
		if err, _ := NewTweetFile(root, all[0], tsrv.user, tsrv.group, 0600, all[1]); err != nil {
			log.Printf("Failed to allocate: %s for %s\n", all[0], all[1])
		}
	}
	return nil
}

func start_service () {
	l := go9p.NewLogger(*logsz)

	tsrv.srv = srv.NewFileSrv(root)
	tsrv.srv.Dotu = true // 9p2000.u
	tsrv.srv.Debuglevel = *debug

	tsrv.srv.Start(tsrv.srv)
	tsrv.srv.Id = "tweetfs"
	tsrv.srv.Log = l

	err := tsrv.srv.StartNetListener("unix", *addr)
	if err != nil {
		log.Panic(err)
	}
}

func main() {
	
	tff := new(TweetFileFactory)
	if err := tff.Add(root, "creator", tsrv.user, tsrv.group, 0600, tff); err != nil {
		log.Panicf("Failed to create the creator: %v\n", err)
	}
	
	if err, _ := NewTweetFile(root, "metadata", tsrv.user, tsrv.group, 0600, METADATA_SFX); err != nil {
		log.Panicf("Failed to allocate metadata endpoint: %v\n", err)
	}

	start_service()
	
	return
}
