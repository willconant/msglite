package msglite

import (
	"net"
	"fmt"
	"strconv"
	"os"
	"bufio"
)

const (
	readyCommandStr   = "R"
	messageCommandStr = "M"
	queryCommandStr   = "Q"
	quitCommandStr    = "X"
	errorCommandStr   = "E"
)

const (
	timeoutHeaderStr  = "m"
	toHeaderStr       = "t"
	replyHeaderStr    = "r"
	bodyLenHeaderStr  = "b"
	errorHeaderStr    = "e"
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
			onAddress, ok := headers[strconv.Itoa(onAddressCount)]
			if !ok {
				break
			}
			
			onAddresses[onAddressCount] = onAddress
			onAddressCount += 1
		}
	
		if onAddressCount == 0 {
			stream.WriteError(os.NewError("missing onAddress header")); return
		}
		
		timeoutStr, ok := headers[timeoutHeaderStr]
		if !ok {
			stream.WriteError(os.NewError("missing timeout header")); return
		}
		
		timeout, err := strconv.Atoi64(timeoutStr)
		if err != nil {
			stream.WriteError(os.NewError("invalid timeout format")); return
		}
		
		msg := <-server.exchange.Ready(onAddresses[0:onAddressCount], timeout)
	
		err = stream.WriteMessage(msg)
		if err != nil {
			stream.WriteError(err); return
		}
	}
	
	handleMessage := func(headers map[string]string) {
		bodyLen, err := strconv.Atoi(headers[bodyLenHeaderStr])
		if err != nil {
			stream.WriteError(os.NewError("invalid body length format")); return
		}
		
		timeout, err := strconv.Atoi64(headers[timeoutHeaderStr])
		if err != nil {
			stream.WriteError(os.NewError("invalid timeout format")); return
		}
		
		body, err := stream.ReadBody(bodyLen)
		
		server.exchange.Send(headers[toHeaderStr], headers[replyHeaderStr], timeout, body)
	}
	
	handleQuery := func(headers map[string]string) {
		timeoutStr, ok := headers[timeoutHeaderStr]
		if !ok {
			stream.WriteError(os.NewError("missing timeout header")); return
		}
		
		timeout, err := strconv.Atoi64(timeoutStr)
		if err != nil {
			stream.WriteError(os.NewError("invalid timeout format")); return
		}
	
		bodyLen, err := strconv.Atoi(headers[bodyLenHeaderStr])
		if err != nil {
			stream.WriteError(os.NewError("invalid body length format")); return
		}
		
		body, err := stream.ReadBody(bodyLen)
		
		msg := server.exchange.Query(headers[toHeaderStr], timeout, body)
		
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
		case readyCommandStr:
			handleReady(headers)
		case messageCommandStr:
			handleMessage(headers)
		case queryCommandStr:
			handleQuery(headers)
		case quitCommandStr:
			stream.Close()	
		default:
			stream.WriteError(os.NewError("invalid command"))
		}
	}
}
