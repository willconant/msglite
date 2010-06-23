// Copyright (c) 2010 William R. Conant, WillConant.com
// Use of this source code is governed by the MIT licence:
// http://www.opensource.org/licenses/mit-license.php

package main

import (
	"msglite"
	"flag"
	"fmt"
	"os"
	"strconv"
)

var client *msglite.Client

func main() {
	var network, laddr string
	flag.StringVar(&network, "network", "unix", "unix or tcp")
	flag.StringVar(&laddr, "address", "", "listen address (either socket path, or ip:port)")
	flag.Parse()
	
	if laddr == "" {
		switch network {
		case "unix":
			laddr = "/tmp/msglite.socket"
		case "tcp":
			laddr = "127.0.0.1:9813"
		}
	}
	
	var err os.Error
	client, err = msglite.NewClient(network, laddr)
	if err != nil {
		panic(err)
	}
	
	switch flag.Arg(0) {
	case "send":
		doSend()
	case "ready":
		doReady()
	case "query":
		doQuery()
	default:
		fmt.Println("command must be one of send, ready, or query")
		os.Exit(1)
	}
}

func doSend() {
	body := flag.Arg(1)
	timeoutStr := flag.Arg(2)
	toAddr := flag.Arg(3)
	var replyAddr = ""
	if flag.NArg() > 3 {
		replyAddr = flag.Arg(4)
	}
	
	timeout, err := strconv.Atoi64(timeoutStr)
	if err != nil {
		panic(err)
	}
	
	err = client.Send(body, timeout, toAddr, replyAddr)
	if err != nil {
		panic(err)
	}
}

func doReady() {
	timeoutStr := flag.Arg(1)
	timeout, err := strconv.Atoi64(timeoutStr)
	if err != nil {
		panic(err)
	}
	
	var addrs [8]string
	addrCount := 0
	for i := 2; i <= flag.NArg(); i++ {
		addrs[i-2] = flag.Arg(i)
		addrCount++
	}
	
	msg, err := client.Ready(timeout, addrs[0:addrCount])
	if err != nil {
		panic(err)
	}
	
	printMsg(msg)
}

func doQuery() {
	body := flag.Arg(1)
	timeoutStr := flag.Arg(2)
	toAddr := flag.Arg(3)
	
	timeout, err := strconv.Atoi64(timeoutStr)
	if err != nil {
		panic(err)
	}
	
	msg, err := client.Query(body, timeout, toAddr)
	if err != nil {
		panic(err)
	}
	
	printMsg(msg)
}

func printMsg(msg *msglite.Message) {
	if msg == nil {
		fmt.Println("*")
		return
	}
	
	fmt.Printf("> %v %v %v %v\n", len(msg.Body), msg.TimeoutSeconds, msg.ToAddress, msg.ReplyAddress)
	if msg.Body != "" {
		fmt.Println(msg.Body)
	}
}
