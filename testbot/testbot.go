package main

import (
	"fmt"

	"github.com/Leimy/rx-go/bot"
)

func main() {
	cout := make(chan string)
	cin := make(chan string)
	go func() {
		for {
			bot.NewBot("#radioxenu", "testbot", "irc.radioxenu.com:6667", cin, cout)
		}
	}()

	for s := range cin {
		fmt.Printf("Got data: %s\n", s)
	}
}
