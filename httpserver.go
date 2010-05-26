// Copyright (c) 2010 William R. Conant, WillConant.com
// Use of this source code is governed by the MIT licence:
// http://www.opensource.org/licenses/mit-license.php

package msglite

import (
	"bytes"
	"fmt"
	"json"
	"net"
	"os"
)

const httpTimeout = 180

type HttpServer struct {
	exchange *Exchange
	exchangeToAddress string
	listener net.Listener
	quitChan chan bool
}

func NewHttpServer(exchange *Exchange, network string, laddr string, exchangeToAddress string) (server *HttpServer) {
	server = new(HttpServer)
	server.exchange = exchange
	server.exchangeToAddress = exchangeToAddress
	server.quitChan = make(chan bool)
	
	var err os.Error
	server.listener, err = net.Listen(network, laddr)
	if err != nil {
		panic(err)
	}
	
	if network == "unix" {
		os.Chmod(laddr, 0777)
	}
	
	return
}

func (server *HttpServer) Run() {
	connChan := make(chan net.Conn)
	
	go func() {
		for {
			conn, err := server.listener.Accept()
			if err != nil {
				panic(fmt.Sprintf("error accepting connection: %v", err))
			}
			
			connChan <- conn
		}
	}()

	for {
		select {
		case netConn := <- connChan:
			go server.handle(netConn)
			
		case <- server.quitChan:
			server.listener.Close()
			return
		}
	}
}

func (server *HttpServer) Quit() {
	server.quitChan <- true
}

func doError(conn net.Conn, err os.Error) {
	var b bytes.Buffer
	b.WriteString("HTTP/1.0 500 Internal Server Error\r\n")
	b.WriteString("Content-Type: text/plain\r\n")
	b.WriteString("Connection: close\r\n\r\n")
	b.WriteString(fmt.Sprintf("An Error Has Occurred\n%v\n", err))
	
	conn.Write(b.Bytes())
	conn.Close()
}

func (server *HttpServer) handle(conn net.Conn) {
	req, err := readHttpRequest(conn)
	replyMsg, err := server.relayRequest(req)
	if err != nil {
		doError(conn, err)
		return
	}
	if replyMsg == nil {
		doError(conn, os.NewError("reply timed out"))
		return
	}
	server.relayReply(replyMsg, conn)
}

func (server *HttpServer) relayRequest(req *httpRequest) (*Message, os.Error) {
	bodyAddr := server.exchange.GenerateUnusedAddress()
	replyAddr := server.exchange.GenerateUnusedAddress()

	envMap := make(map[string]interface{})
	envMap["method"] = req.method
	envMap["url"] = req.url
	envMap["protocol"] = req.protocol
	envMap["headers"] = req.headers
	envMap["bodyAddr"] = bodyAddr
	
	json, err := json.Marshal(envMap)
	if err != nil {
		panic(err)
	}
	
	server.exchange.Send(string(json), httpTimeout, server.exchangeToAddress, replyAddr)
	server.exchange.Send(string(req.body), httpTimeout, bodyAddr, "")
	
	return server.exchange.Ready(httpTimeout, []string{replyAddr}), nil
}

func (server *HttpServer) relayReply(replyMsg *Message, conn net.Conn) {
	var err os.Error
	var reply []interface{}
		
	err = json.Unmarshal([]byte(replyMsg.Body), &reply)
	if err != nil {
		goto BadReply
	}
	
	status, ok := reply[0].(string)
	if !ok {
		err = os.NewError("first item of reply wasn't a string")
		goto BadReply
	}
	
	st, ok := statusText[status]
	if !ok {
		err = &badStringError{"not a valid status code", status}
		goto BadReply
	}
		
	headers, ok := reply[1].([]interface{})
	if !ok {
		err = os.NewError("second item of reply wasn't a []interface{}")
		goto BadReply
	}
	
	if len(headers) % 2 != 0 {
		err = os.NewError("headers array had odd number of items")
		goto BadReply
	}
	
	var respBuffer bytes.Buffer
	respBuffer.WriteString("HTTP/1.0 " + status + " " + st + "\r\n")
	
	for i := 0; i < len(headers); i += 2 {
		kStr, ok := headers[i].(string)
		if !ok {
			err = os.NewError("header key wasn't a string")
			goto BadReply
		}
		
		kStr = canonicalHeaderKey(kStr)
		
		vStr, ok := headers[i+1].(string)
		if !ok {
			err = os.NewError("header value wasn't a string")
			goto BadReply
		}
		
		// there is almost certainly a whole list of headers I should ignore from the worker
		if kStr != "Connection" {
			respBuffer.WriteString(kStr + ": " + vStr + "\r\n")
		}
	}
			
	respBuffer.WriteString("Connection: close\r\n\r\n")
	
	_, err = conn.Write(respBuffer.Bytes())
	if err != nil {
		// couldn't do the actual writing, just give up
		conn.Close()
		return
	}
	
	for {
		switch replyBodyMsg := server.exchange.Ready(httpTimeout, []string{replyMsg.ToAddress}); {
		case replyBodyMsg == nil:
			// we really expected a message here... this is busted
			conn.Close()
			return
		case replyBodyMsg.Body == "":
			// an empty message indicates we're all done
			conn.Close()
			return
		default:
			_, err = conn.Write([]byte(replyBodyMsg.Body))
			if err != nil {
				conn.Close()
				return
			}
		}
	}
		
BadReply:
	doError(conn, err)
}
