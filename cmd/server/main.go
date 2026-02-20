package main

import (
	"log"

	"github.com/shashiranjanraj/kashvi/internal/server"
)

func main() {
	if err := server.Start(); err != nil {
		log.Fatal(err)
	}
}
