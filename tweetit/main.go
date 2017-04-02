package main

import (
	"bufio"
	"github.com/Leimy/rx-go/twit"
	"os"
)

func condTweet(line string, err error) func(twit.Tweeter) {
	if err == nil {
		return func(t twit.Tweeter) {
			t(line)
		}
	}
	return func(_ twit.Tweeter) {}
}

func main() {
	condTweet(bufio.NewReader(os.Stdin).ReadString('\n'))(twit.MakeTweeter("http://tunein.com/radio/Radio-Xenu-s118981/ @radioxenu"))
}
