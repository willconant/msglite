package main

import (
	"./msglite"
	"flag"
	"fmt"
	"os/signal"
)

func main() {
	var network, laddr string
	flag.StringVar(&network, "n", "unix", "unix or tcp")
	flag.StringVar(&laddr, "a", "", "listen address (either socket path, or ip:port)")
	flag.Parse()
	
	if laddr == "" {
		switch network {
		case "unix":
			laddr = "/tmp/msglite.socket"
		case "tcp":
			laddr = "127.0.0.1:9999"
		}
	}	

	server := msglite.NewServer(msglite.NewExchange(), network, laddr)
	
	go func() {
		quit := false
		for !quit {
			sig := <-signal.Incoming
			fmt.Printf("received signal %v\n", sig)
			switch sig.(signal.UnixSignal) {
			case 1: // SIGHUP
				quit = true
			case 2: // SIGNINT
				quit = true
			case 3: // SIGQUIT
				quit = true
			case 15: // SIGTERM
				quit = true
			}
		}
		server.Quit()
	}()
	
	fmt.Printf("msglite listening on %v (%v)\n", laddr, network)
	server.Run()
	fmt.Printf("msglite quitting\n")
}
