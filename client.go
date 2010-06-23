// Copyright (c) 2010 William R. Conant, WillConant.com
// Use of this source code is governed by the MIT licence:
// http://www.opensource.org/licenses/mit-license.php

package msglite

import (
	"net"
	"strconv"
	"os"
	"bufio"
)

type Client struct {
	conn net.Conn
	stream *CommandStream
}

func NewClient(network string, laddr string) (*Client, os.Error) {
	conn, err := net.Dial(network, "", laddr)
	if err != nil {
		return nil, err
	}
	
	client := &Client{
		conn,
		&CommandStream{bufio.NewReader(conn), conn, false},
	}
	
	return client, nil
}

func (client *Client) Send(body string, timeoutSeconds int64, toAddress string, replyAddress string) os.Error {
	return client.stream.WriteMessage(&Message{toAddress, replyAddress, timeoutSeconds, body, 0})
}

func (client *Client) Ready(timeoutSeconds int64, onAddresses []string) (*Message, os.Error) {
	outCommand := make([]string, len(onAddresses) + 2)
	outCommand[0] = readyCommandStr
	outCommand[1] = strconv.Itoa64(timeoutSeconds)
	for i := 0; i < len(onAddresses); i++ {
		outCommand[i+2] = onAddresses[i]
	}
	
	err := client.stream.WriteCommand(outCommand)
	if err != nil {
		return nil, err
	}
	
	return client.stream.ReadMessage()
}

func (client *Client) Query(body string, timeoutSeconds int64, toAddress string) (*Message, os.Error) {
	err := client.stream.WriteQuery(body, timeoutSeconds, toAddress)
	if err != nil {
		return nil, err
	}
	
	return client.stream.ReadMessage()
}

func (client *Client) Quit() os.Error {
	err := client.stream.WriteQuit()
	client.conn.Close()
	return err
}
