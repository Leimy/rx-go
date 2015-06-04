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
	"fmt"
	"errors"
)

// Describes the file system we serve up to host
// Twitter capabilities for the metabot
type Twitfs struct {
	srv   *srv.Fsrv
	user  go9p.User
	group go9p.Group
}

// Describes an individual file we serve in the
// file system.
type TweetFile struct {
	srv.File
	data []byte
	tweeter twit.Tweeter
}

// A data structure to track the state of the
// "creator" file which exists to set up other files.
type TweetFileFactory struct {
	srv.File
	data []byte
}

// The common functions that are involved with creating
// a regular tweeting endpoint.
func NewTweetFile(name string, user go9p.User, group go9p.Group, mode uint32, template string) (error, *TweetFile) {
	twitterentry := new(TweetFile)
	twitterentry.tweeter = twit.MakeTweeter(template)
	err := twitterentry.Add(root, name, user, group, mode, twitterentry)
	if err != nil {
		return err, nil
	}
	return nil, twitterentry
}

// Just a constant for metadata
const METADATA_SFX string = "on @radioxenu http://tunein.com/radio/Radio-Xenu-s118981/"

// Command line argument parsing and defaults
var addr = flag.String("a", "./crustysock", "unix domain socket path")
var debug = flag.Int("d", 0, "debuglevel")
var logsz = flag.Int("l", 2048, "log size")
var tsrv Twitfs
var root *srv.File

// Runs one time when the module loads... initializes stuff (hence the name)
func init () {
	flag.Parse()
	tsrv.user = go9p.OsUsers.Uid2User(os.Geteuid())
	tsrv.group = go9p.OsUsers.Gid2Group(os.Getegid())
	root = new(srv.File)
	if err := root.Add(nil, "/", tsrv.user, nil, go9p.DMDIR|0555, nil); err != nil {
		log.Panic(err)
	}
}

// When a client writes to a tweet file, we capture the bytes, and handle them
// in Clunk
func (t *TweetFile) Write(fid *srv.FFid, buf []byte, offset uint64) (int, error) {
	t.data = append(t.data, buf...)
	return len(buf), nil
}

// When the client decides it's done with this TweetFile, we run this action
// It run the set up tweeter function on the stringified form of the bytes
// it has collected.
func (t *TweetFile) Clunk(fid *srv.FFid) error {
	go func () {
		if err := t.tweeter(string(t.data)); err != nil {
			log.Printf("Error tweeting: %s\n", err)
		}
	}()
	log.Printf("Clunk: %p\n", fid)
	t.data = []byte{}
	return nil
}

// This simply says "we'll allow you to delete this file"
func (t *TweetFile) Remove(fid *srv.FFid) error {
	return nil
}

// When someone writes to the creator file, we capture the bytes
// we'll deal with them in Clunk
func (tff *TweetFileFactory) Write(fid *srv.FFid, buf []byte, offset uint64) (int, error) {
	tff.data = append(tff.data, buf...)
	return len(buf), nil
}

// When the creator is Clunk'd by the client, we try to process the formatted string
// newfilename|suffix string for tweet
// And if successful, you get a new TweetFile you can write to that appends "suffix string for tweet"
// to the message before sending to twitter.
func (tff *TweetFileFactory) Clunk(fid *srv.FFid) (err error) {
	s := string(tff.data)
	all := strings.SplitN(s, "|", 2)

	if len(all) != 2 {
		s := fmt.Sprintf("Illegal reqeust, ignoring: %s", s)
		log.Printf("%s\n", s)
		err = errors.New(s)
	} else {
		if err, _ := NewTweetFile(all[0], tsrv.user, tsrv.group, 0600, all[1]); err != nil {
			log.Printf("Failed to allocate: %s for %s\n", all[0], all[1])
		}
	}
	tff.data = []byte{}
	return err
}


func start_service () {
	l := go9p.NewLogger(*logsz)

	tsrv.srv = srv.NewFileSrv(root)
	tsrv.srv.Dotu = true // 9p2000.u
	tsrv.srv.Debuglevel = *debug

	tsrv.srv.Start(tsrv.srv)
	tsrv.srv.Id = "tweetfs"
	tsrv.srv.Log = l

	os.Remove(*addr)

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
	
	if err, _ := NewTweetFile("metadata", tsrv.user, tsrv.group, 0600, METADATA_SFX); err != nil {
		log.Panicf("Failed to allocate metadata endpoint: %v\n", err)
	}

	start_service()
	
	return
}
