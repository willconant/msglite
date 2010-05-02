package msglite

import (
	"net"
	"fmt"
	"strconv"
	"os"
	"io"
	"bufio"
)

type SocketServer struct {
	exchange *Exchange
	listener net.Listener
	quitChan chan bool
}

func NewSocketServer(exchange *Exchange, path string) (server *SocketServer) {
	server = new(SocketServer)
	server.exchange = exchange
	server.quitChan = make(chan bool)
	
	var err os.Error
	
	server.listener, err = net.Listen("unix", path)
	if err != nil {
		panic(fmt.Sprintf("couldn't listen at %v: %v", path, err))
	}
	
	return
}

func (server *SocketServer) Run() {
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
		case conn := <- connChan:
			go server.handleConn(conn)
			
		case <- server.quitChan:
			server.listener.Close()
			return
		}
	}
}

func (server *SocketServer) handleConn(conn net.Conn) {
	readWriter := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
	
	connErr:= func(err os.Error) {
		panic(fmt.Sprintf("error communicating with client: %v", err))
	}
	
	readline := func() string {
		line, err := readWriter.ReadString('\n')
		if err != nil {
			connErr(err)
		}
		return line[0:len(line)-1]
	}
	
	writeline := func(s string) {
		_, err := readWriter.WriteString(s + "\n")
		if err != nil {
			connErr(err)
		}
		readWriter.Flush()
	}
	
	readBody := func() (string, bool) {
		bodyLen, err := strconv.Atoi(readline())
		if err != nil {
			writeline("ERROR")
			writeline("invalid value for body length")
			return "", true
		}
		
		bodyBuf := make([]byte, bodyLen)
		_, err = io.ReadAtLeast(readWriter, bodyBuf, bodyLen)
		if err != nil {
			connErr(err)
		}
		readline()
		
		return string(bodyBuf), false
	}
	
	writeMessage := func(msg Message) {
		writeline("MESSAGE")
		writeline(msg.ToAddress)
		writeline(msg.ReplyAddress)
		writeline(strconv.Btoa(msg.Broadcast))
		writeline(strconv.Itoa(len(msg.Body)))
		writeline(msg.Body)
	}

	for {
		switch readline() {
		case "READY":
			onAddress := readline()
			msg := <- server.exchange.ReadyOnAddress(onAddress)
			writeMessage(msg)
			
		case "MESSAGE":
			toAddress := readline()
			replyAddress := readline()
			
			broadcast, err := strconv.Atob(readline())
			if err != nil {
				writeline("ERROR")
				writeline("invalid value for broadcast")
				break
			}
			
			body, brk := readBody()
			if brk {
				break
			}
			
			server.exchange.SendMessage(toAddress, replyAddress, broadcast, body)
		
		case "QUERY":
			toAddress := readline()
			body, brk := readBody()
			if brk {
				break
			}
			msg := server.exchange.SendQuery(toAddress, body)
			writeMessage(msg)
			
		case "CLOSE":
			err := conn.Close()
			if err != nil {
				connErr(err)
			}
			return
		
		default:
			writeline("ERROR")
			writeline("invalid command")
		}
	}
}
