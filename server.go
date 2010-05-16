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
	
	if network == "unix" {
		os.Chmod(laddr, 0777)
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
		var onAddresses [maxOnAddresses]string
		onAddressCount := 0
		
		for onAddressCount < maxOnAddresses {
			onAddress, ok := headers["on" + strconv.Itoa(onAddressCount)]
			if !ok {
				break
			}
			
			onAddresses[onAddressCount] = onAddress
			onAddressCount += 1
		}
	
		if onAddressCount == 0 {
			stream.WriteError(os.NewError("missing header in READY: on0..onN")); return
		}
		
		timeoutStr, ok := headers["timeout"]
		if !ok {
			stream.WriteError(os.NewError("missing header in READY: timeout")); return
		}
		
		timeout, err := strconv.Atoi64(timeoutStr)
		if err != nil {
			stream.WriteError(os.NewError("invalid format for timeout header in READY")); return
		}
		
		msg := <-server.exchange.Ready(onAddresses[0:onAddressCount], timeout)
	
		err = stream.WriteMessage(msg)
		if err != nil {
			stream.WriteError(err); return
		}
	}
	
	handleMessage := func(headers map[string]string) {
		bodyLen, err := strconv.Atoi(headers["body"])
		if err != nil {
			stream.WriteError(os.NewError("invalid format for header body")); return
		}
		
		timeout, err := strconv.Atoi64(headers["timeout"])
		if err != nil {
			stream.WriteError(os.NewError("invalid format for header timeout")); return
		}
		
		body, err := stream.ReadBody(bodyLen)
		
		server.exchange.Send(headers["to"], headers["reply"], timeout, body)
	}
	
	handleQuery := func(headers map[string]string) {
		timeoutStr, ok := headers["timeout"]
		if !ok {
			stream.WriteError(os.NewError("missing header in READY: timeout")); return
		}
		
		timeout, err := strconv.Atoi64(timeoutStr)
		if err != nil {
			stream.WriteError(os.NewError("invalid format for timeout header in READY")); return
		}
	
		bodyLen, err := strconv.Atoi(headers["body"])
		if err != nil {
			stream.WriteError(os.NewError("invalid format for header body")); return
		}
		
		body, err := stream.ReadBody(bodyLen)
		
		msg := server.exchange.Query(headers["to"], timeout, body)
		
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
