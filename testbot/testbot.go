package main

import (
	"fmt"

	"github.com/Leimy/rx-go/bot"
)

func main() {
	cout := make(chan string)
	cin := make(chan string)
	go func() {
		if err := bot.NewBot("#radioxenu", "testbot", "irc.radioxenu.com:6667", cout, cin); err != nil {
			panic(err)
		}
	}()

	for s := range cout {
		fmt.Printf("Got data: %s\n", s)
	}
}
