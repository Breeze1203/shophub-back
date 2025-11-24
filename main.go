package main

import (
	"LiteAdmin/server"
)

func main() {
	s := server.NewServer()
	s.Start(":8080")
}
