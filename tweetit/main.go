package main

import "github.com/Leimy/rx-go/twit"
import "os"

func main () {
	twit.MakeTweeter("http://tunein.com/radio/Radio-Xenu-s118981/ @radioxenu")(os.Args[1])
}
