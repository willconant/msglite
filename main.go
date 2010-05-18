package main

import (
	"msglite"
	"flag"
	"fmt"
	"os"
	"os/signal"
)

func main() {
	var network, laddr, logLevel string
	flag.StringVar(&network, "network", "unix", "unix or tcp")
	flag.StringVar(&laddr, "laddr", "", "listen address (either socket path, or ip:port)")
	flag.StringVar(&logLevel, "loglevel", "info", "logging level (one of 'minimal', 'info' or 'debug')")
	flag.Parse()
	
	if laddr == "" {
		switch network {
		case "unix":
			laddr = "/tmp/msglite.socket"
		case "tcp":
			laddr = "127.0.0.1:9999"
		}
	}	
	
	exchange := msglite.NewExchange()
	
	switch logLevel {
	case "minimal":
		exchange.SetLogLevel(msglite.LogLevelMinimal)
	case "info":
		exchange.SetLogLevel(msglite.LogLevelInfo)
	case "debug":
		exchange.SetLogLevel(msglite.LogLevelDebug)
	default:
		os.Stderr.WriteString(fmt.Sprintf("linvalid log level: %v\n", logLevel))
		flag.PrintDefaults()
		os.Exit(1)
	}
	
	server := msglite.NewServer(exchange, network, laddr)
	
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
