package main

import (
	"github.com/Leimy/rx-go/bot"
	"fmt"
)

func main() {
	c := make(chan string)
	go func () {
		if err := bot.NewBot("#radioxenu", "testbot", "irc.radioxenu.com:6667", c); err != nil {
			panic(err)
		}
	}()

	for s := range c {
		fmt.Printf("Got data: %s\n", s)
	}
}
