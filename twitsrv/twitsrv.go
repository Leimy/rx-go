// twitter service for 9p

package main

import (
	"flag"
	"github.com/Leimy/rx-go/twit"
	"github.com/mortdeus/go9p"
	"github.com/mortdeus/go9p/srv"
	"log"
	"os"
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

var addr = flag.String("a", "./crustysock", "unix domain socket path")
var debug = flag.Int("d", 0, "debuglevel")
var logsz = flag.Int("l", 2048, "log size")
var tsrv Twitfs


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

func main() {
	flag.Parse()

	tsrv.user = go9p.OsUsers.Uid2User(os.Geteuid())
	tsrv.group = go9p.OsUsers.Gid2Group(os.Getegid())

	root := new(srv.File)
	err := root.Add(nil, "/", tsrv.user, nil, go9p.DMDIR|0555, nil)
	if err != nil {
		log.Panic(err)
	}

	twitterentry := new(TweetFile)
	twitterentry.tweeter = twit.MakeTweeter("on @radioxenu http://tunein.com/radio/Radio-Xenu-s118981/")
	
	err = twitterentry.Add(root, "tweet", go9p.OsUsers.Uid2User(os.Geteuid()), nil, 0600, twitterentry)
	if err != nil {
		log.Panic(err)
	}

	l := go9p.NewLogger(*logsz)

	tsrv.srv = srv.NewFileSrv(root)
	tsrv.srv.Dotu = true // 9p2000.u
	tsrv.srv.Debuglevel = *debug

	tsrv.srv.Start(tsrv.srv)
	tsrv.srv.Id = "tweetfs"
	tsrv.srv.Log = l

	err = tsrv.srv.StartNetListener("unix", *addr)
	if err != nil {
		log.Panic(err)
	}

	return
}
