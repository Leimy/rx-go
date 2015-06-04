// metadata server service
package main

import (
	"flag"
	"github.com/Leimy/rx-go/meta"
	"github.com/mortdeus/go9p"
	"github.com/mortdeus/go9p/srv"
	"log"
	"os"
	"errors"
)


var streamaddr = flag.String("s", "http://radioxenu.com:8000/relay", "Music Stream URL")
var addr = flag.String("a", "./metasock", "unix domain socket path")
var debug = flag.Int("d", 0, "debuglevel")
var logsz = flag.Int("l", 2048, "log size")
var msrv Metafs
var root *srv.File

func init () {
	flag.Parse()
	msrv.user = go9p.OsUsers.Uid2User(os.Geteuid())
	msrv.group = go9p.OsUsers.Gid2Group(os.Getegid())
	root = new(srv.File)
	if err := root.Add(nil, "/", msrv.user, nil, go9p.DMDIR|0555, nil); err != nil {
		log.Panic(err)
	}
}


// Describes the file system interface itself
type Metafs struct {
	srv *srv.Fsrv
	user go9p.User
	group go9p.Group
}

// Read-only file for accessing metadata
type MetaFile struct {
	srv.File
	S *service
}

// Tiny service tracking the current state of the world.
type service struct {
	s string
	e error
}

func new_service (mf *MetaFile) (*service,  error) {
	s := new(service)
	s.s = ""
	
	c, err := meta.StreamMeta(*streamaddr)
	if err != nil {
		return nil, err
	}
	go func () {
		
		for cur := range c {
			s.s = cur
			log.Printf("%s\n", s.s)
			func() {
				mf.Lock()
				defer mf.Unlock()
				mf.Length = uint64(len(s.s))
			}()
		}
		s.e = errors.New("Metadata stream terminated")
	}()
	
	return s, nil
}

func (s *service) last () string {
	return s.s
}

func (s *service) err () error {
	return s.e
}


func (m *MetaFile) Read(fid *srv.FFid, buf []byte, offset uint64) (int, error) {
	m.Lock()
	defer m.Unlock()

	if offset > m.Length {
		return 0, nil
	}

	log.Printf("Request to read: %v %v\n", len(buf), offset)
	// check status of the metadata source
	err := m.S.err()
	if m.S.err() != nil {
		// if it's busted, try to re-initialize it
		if m.S, err = new_service(m); err != nil {
			// or fail
			return 0, err
		}

	}

	// get last string
	lastsong := []byte(m.S.last())
	
	return copy(buf, lastsong[int(offset):]) , nil
}

func serve () {
	l := go9p.NewLogger(*logsz)

	msrv.srv = srv.NewFileSrv(root)
	msrv.srv.Dotu = true //9p2000.u
	msrv.srv.Debuglevel = *debug

	msrv.srv.Start(msrv.srv)
	msrv.srv.Id = "metafs"
	msrv.srv.Log = l
	
	// rm unix socket
	os.Remove(*addr)

	if err := msrv.srv.StartNetListener("unix", *addr); err != nil {
		log.Panic(err)
	}
}	

func main () {
	entry := new(MetaFile)
	var err error
	if entry.S, err = new_service(entry); err != nil {
		log.Panic(err)
	}

	if err := entry.Add(root, "metadata", msrv.user, msrv.group, 0400, entry); err != nil {
		log.Panicf("Failed to create metadata: %v\n", err)
	}
	

	serve()

	return
}
