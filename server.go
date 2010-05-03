package msglite

import (
	"net"
	"fmt"
	"strconv"
	"os"
	"bufio"
)

type Server struct {
	exchange *Exchange
	listener net.Listener
	quitChan chan bool
}

func NewServer(exchange *Exchange, network string, laddr string) (server *Server) {
	server = new(Server)
	server.exchange = exchange
	server.quitChan = make(chan bool)
	
	var err os.Error
	server.listener, err = net.Listen(network, laddr)
	if err != nil {
		panic(err)
	}
	
	return
}

func (server *Server) Run() {
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
			go server.handle(&CommandStream{bufio.NewReader(netConn), netConn, false})
			
		case <- server.quitChan:
			server.listener.Close()
			return
		}
	}
}

func (server *Server) Quit() {
	server.quitChan <- true
}

func (server *Server) handle(stream *CommandStream) {
		
	handleReady := func(headers map[string]string) {
		onAddress, ok := headers["on"]
		if !ok {
			stream.WriteError(os.NewError("missing header in READY: on"))
		}
		
		msg := <- server.exchange.ReadyOnAddress(onAddress)
	
		err := stream.WriteMessage(msg)
		if err != nil {
			stream.WriteError(err); return
		}
	}
	
	handleMessage := func(headers map[string]string) {
		bodyLen, err := strconv.Atoi(headers["body"])
		if err != nil {
			stream.WriteError(os.NewError("invalid format for header body")); return
		}
		
		body, err := stream.ReadBody(bodyLen)
		
		server.exchange.SendMessage(headers["to"], headers["reply"], headers["bcast"] == "1", body)
	}
	
	handleQuery := func(headers map[string]string) {
		bodyLen, err := strconv.Atoi(headers["body"])
		if err != nil {
			stream.WriteError(os.NewError("invalid format for header body")); return
		}
		
		body, err := stream.ReadBody(bodyLen)
		
		msg := server.exchange.SendQuery(headers["to"], body)
		
		err = stream.WriteMessage(msg)
		if err != nil {
			stream.WriteError(err); return
		}
	}

	for !stream.closed {
		command, headers, err := stream.ReadCommand()
		if err != nil {
			stream.WriteError(err)
			break
		}
		
		switch command {
		case "READY":
			handleReady(headers)
		case "MESSAGE":
			handleMessage(headers)
		case "QUERY":
			handleQuery(headers)
		case "QUIT":
			stream.Close()	
		default:
			stream.WriteError(os.NewError("invalid command"))
		}
	}
}
