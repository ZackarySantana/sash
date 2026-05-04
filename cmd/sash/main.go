package main

import (
	"log"

	"github.com/zackarysantana/sash/src/cli"
)

func main() {
	if err := cli.Run(); err != nil {
		log.Fatal(err)
	}
}
