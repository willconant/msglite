// Copyright (c) 2010 William R. Conant, WillConant.com
// Use of this source code is governed by the MIT licence:
// http://www.opensource.org/licenses/mit-license.php

package main

import (
	"msglite"
	"flag"
	"fmt"
	"os"
	"os/signal"
)

func main() {
	var network, laddr, httpNetwork, httpLaddr, httpReqMsgAddr, logLevel string
	flag.StringVar(&network, "network", "unix", "unix or tcp")
	flag.StringVar(&laddr, "address", "", "listen address (either socket path, or ip:port)")
	flag.StringVar(&httpNetwork, "http-network", "tcp", "unix or tcp")
	flag.StringVar(&httpLaddr, "http-address", "", "http listen address (either socket path, or ip:port)")
	flag.StringVar(&httpReqMsgAddr, "http-msg-address", "msglite.httpRequests", "msglite address to which http request messages are sent")
	flag.StringVar(&logLevel, "loglevel", "info", "logging level (one of 'minimal', 'info' or 'debug')")
	flag.Parse()
	
	if laddr == "" {
		switch network {
		case "unix":
			laddr = "/tmp/msglite.socket"
		case "tcp":
			laddr = "127.0.0.1:9813"
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
	fmt.Printf("msglite listening on %v (%v)\n", laddr, network)
	
	var httpServer *msglite.HttpServer
	if httpLaddr != "" {
		httpServer = msglite.NewHttpServer(exchange, httpNetwork, httpLaddr, httpReqMsgAddr)
		go httpServer.Run()
		fmt.Printf("msglite http server listening on %v (%v) requests are bing routed to %v\n", httpLaddr, httpNetwork, httpReqMsgAddr)
	}

	go func() {
		quit := false
		for !quit {
			sig := <-signal.Incoming
			fmt.Printf("received signal %v\n", sig)
			switch sig.(signal.UnixSignal) {
			case 1: // SIGHUP
				quit = true
			case 2: // SIGINT
				quit = true
			case 3: // SIGQUIT
				quit = true
			case 15: // SIGTERM
				quit = true
			}
		}
		if httpServer != nil {
			httpServer.Quit()
		}
		server.Quit()
	}()
	
	server.Run()
	fmt.Printf("msglite quitting\n")
}
