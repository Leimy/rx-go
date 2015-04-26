// twitter service for 9p

package main

import (
	"github.com/mortdeus/go9p"
	"github.com/mortdeus/go9p/srv"
	"flag"
	"log"
	"os"
	twit "github.com/Leimy/icy-metago/twitter"
)
type Twitfs struct {
	srv *srv.Fsrv
	user go9p.User
	group go9p.Group
	blksz int
	blkchan chan []byte
	zero []byte // 0s
}

type TweetFile struct {
	srv.File
	data [][]byte
}

var addr = flag.String("addr", "./crustysock", "unix domain socket path")
var debug = flag.Int("d", 0, "debuglevel")
var blksize = flag.Int("b", 200, "block size") // likely enough, twitter y'know
var logsz = flag.Int("l", 2048, "log size")
var tsrv Twitfs

func (t *TweetFile) expand(sz uint64) {
	blknum := sz / uint64(tsrv.blksz)
	if sz & uint64(tsrv.blksz) != 0 {
		blknum++
	}

	data :=make([][]byte, blknum)
	if t.data != nil {
		copy(data, t.data)
	}
	t.data = data
	t.Length = sz
}

// Ripped off from ramfs
func (t *TweetFile) Write(fid *srv.FFid, buf []byte, offset uint64) (int, error) {
	go twit.Tweet(string(buf))
	
	return len(buf), nil
}

// func (t *TweetFile) Clunk(req *srv.Req) {
// defer t.Unlock()
// 	var tweetbody []byte
// 	for _, chunk := range t.data {
// 		tweetbody = append(tweetbody, string(chunk)...)
// 	}
// 	twit.Tweet(string(tweetbody))
// 	req.RespondRclunk()
// }

func main () {
	flag.Parse()
	
	tsrv.user = go9p.OsUsers.Uid2User(os.Geteuid())
	tsrv.group = go9p.OsUsers.Gid2Group(os.Getegid())
	tsrv.blksz = *blksize
	tsrv.blkchan = make(chan []byte, 200)
	tsrv.zero = make([]byte, tsrv.blksz)
	
	root := new(srv.File)
	err := root.Add(nil, "/", tsrv.user, nil, go9p.DMDIR|0555, nil)
	if err != nil {
		log.Panic(err)
	}

	twitterentry := new(TweetFile)
	err = twitterentry.Add(root, "twat", go9p.OsUsers.Uid2User(os.Geteuid()), nil, 0600, twitterentry)
	if err != nil {
		log.Panic(err)
	}
	
	l := go9p.NewLogger(*logsz)
	
	tsrv.srv = srv.NewFileSrv(root)
	tsrv.srv.Dotu = true // 9p2000
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
