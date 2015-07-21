package bot

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"regexp"
	"strings"
)

// The Bot type.  Only thing exported from this module.
type Bot struct {
	*bufio.Reader
	*bufio.Writer
	room          string
	name          string
	serverAndPort string
	linesOut      chan<- string
	linesIn       <-chan string
}

var userRegExp *regexp.Regexp
var actionRegExp *regexp.Regexp
var chanAndMessageRegExp *regexp.Regexp

func init() {
	// TODO: THIS IS MOSTLY CRAP... DO IT AGAIN LATER!
	// if this matches it produces string slices size 6
	// 0 is the whole string that matched
	// 1 is the nickname
	// 2 is the channel involved
	// 3 If it was an action, this is the string "ACTION " (trailing space intentional)
	// 4 Color
	// 5 The message delivered by the nick on the channel
	//	chanAndMessageRegExp = regexp.MustCompile("^:(.+)!.*PRIVMSG (#.+) :(ACTION )?(.+)$")
	chanAndMessageRegExp = regexp.MustCompile("^:([[:print:]]+)!.*PRIVMSG (#[[:print:]]+) :[0-9]*(ACTION )?[^[:digit:]]*?([[:print:]]+)$")
}

// Just some setup stuff for getting into the channel
func (b *Bot) loginstuff() {
	fmt.Fprintf(b, "NICK %s\r\n", b.name)
	fmt.Fprintf(b, "USER %s 0 * :tutorial bot\r\n", b.name)
	fmt.Fprintf(b, "JOIN %s\r\n", b.room)
	if err := b.Flush(); err != nil {
		log.Panic(err)
	}
}

// Filter returns a new slice holding only
// the elements of s that satisfy f()
// Tiny state machine to filter out colors if they're
// encoded.
func filterPrintable(s []byte) []byte {
	var p []byte // == nil
	found := false
	for _, v := range s {
		if !found {
			if v != 3 { // weird color encoding thing
				p = append(p, v)
			} else {
				found = true
			}
		} else {
			found = false
			continue
		}
	}
	return p
}

func (b *Bot) fromIRC(completeSChan chan<- string) {
	defer close(completeSChan)
	scanner := bufio.NewScanner(b)
	for scanner.Scan() {
		completeSChan <- string(filterPrintable([]byte(scanner.Text())))
	}
}

func (b *Bot) parseTokens(lines []string) string {
	if len(lines) == 0 {
		// this is ok, just pass
		return ""
	}
	if len(lines) < 5 {
		log.Panic(lines)
	}

	body := lines[4]

	log.Printf("4 == %q\n", body)

	return body
}

// process each line
func (b *Bot) procLine(line string) {

	// Handle PING so we don't get hung up on.
	if strings.HasPrefix(line, "PING :") {
		resp := strings.Replace(line, "PING", "PONG", 1)
		fmt.Fprintf(b, "%s\r\n", resp)
	} else {
		s := b.parseTokens(chanAndMessageRegExp.FindStringSubmatch(line))
		if s != "" {
			b.linesOut <- s
		}
	}
	if err := b.Flush(); err != nil {
		log.Panic(err)
	}
}

func (b *Bot) loop() {
	completeSChan := make(chan string)

	// Receives lines, dropping things we don't
	go b.fromIRC(completeSChan)

	lchan := make(chan string)
	defer func() {
		close(lchan)
	}()

	// process lines asynchronously from
	// receiving them.
	go func() {
		for line := range lchan {
			// handle ping/pong and other processing
			b.procLine(line)
		}
	}()

	for {
		select {
		case line := <-completeSChan:
			if line == "" {
				return // does exit and cleanup
			}
			lchan <- line // feed lines to processor
		}
	}
}

func bot(room, name, serverAndport string, linesOut chan<- string, linesIn <-chan string) error {
	defer close(linesOut)
	log.Printf("IRC bot connecting to %s as %s to channel %s\n",
		serverAndport, name, room)
	conn, err := net.Dial("tcp4", serverAndport)
	if err != nil {
		return err
	}
	log.Print("Done connecting")

	bot := &Bot{
		bufio.NewReader(conn),
		bufio.NewWriter(conn),
		room,
		name,
		serverAndport,
		linesOut,
		linesIn}

	bot.loginstuff()
	bot.loop()

	return nil
}

// NewBot Doesn't return until the bot loop terminates or crashes
func NewBot(room, name, serverAndPort string, linesIn chan<- string, linesOut <-chan string) error {
	return bot(room, name, serverAndPort, linesIn, linesOut)
}
