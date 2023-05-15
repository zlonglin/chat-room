package main

import (
	"chatroom/cache"
	"chatroom/server"
	"log"
)

func main() {
	cache.Init()
	defer cache.Close()

	s := server.NewServer()
	log.Fatal(s.Run())
}
