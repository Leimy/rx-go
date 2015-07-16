package main

import (
	"fmt"

	"github.com/Leimy/rx-go/meta"
)

func main() {
	ch, err := meta.StreamMeta("http://radioxenu.com:8000/relay")
	if err != nil {
		panic(err)
	}

	for s := range ch {
		fmt.Print(s, "\n")
	}
}
