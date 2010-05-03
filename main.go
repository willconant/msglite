package main

import (
	"./msglite"
	"os/signal"
)

func main() {
	server := msglite.NewServer(msglite.NewExchange(), "unix", "/tmp/msglite.socket")
	
	go func() {
		<-signal.Incoming
		server.Quit()
	}()
	
	server.Run()
}
