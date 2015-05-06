package main

import (
	"github.com/Leimy/rx-go/meta"
	"fmt"
)

func main () {
	ch, err := meta.StreamMeta("http://radioxenu.com:8000")
	if err != nil {
		panic(err)
	}

	for s := range ch {
		fmt.Print(s, "\n")
	}
}
