package msglite

import (
	"net"
	"fmt"
	"strconv"
	"os"
	"bufio"
)

const (
	readyCommandStr   = "<"
	messageCommandStr = ">"
	queryCommandStr   = "?"
	timeoutCommandStr = "*"
	quitCommandStr    = "."
	errorCommandStr   = "-"
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
		
	handleReady := func(params []string) {
		if len(params) < 2 {
			stream.WriteError(os.NewError("ready format: < timeout onAddr1 [onAddr2..onAddrN]")); return
		}
	
		timeout, err := strconv.Atoi64(params[0])
		if err != nil {
			stream.WriteError(os.NewError("invalid timeout format")); return
		}
		
		msg := <-server.exchange.Ready(params[1:], timeout)
	
		err = stream.WriteMessage(msg)
		if err != nil {
			stream.WriteError(err); return
		}
	}
	
	handleMessage := func(params []string) {
		if len(params) < 3 || len(params) > 4 {
			stream.WriteError(os.NewError("message format: > bodyLen timeout toAddr [replyAddr]")); return
		}
	
		bodyLen, err := strconv.Atoi(params[0])
		if err != nil {
			stream.WriteError(os.NewError("invalid body length format")); return
		}
		
		timeout, err := strconv.Atoi64(params[1])
		if err != nil {
			stream.WriteError(os.NewError("invalid timeout format")); return
		}
		
		toAddr := params[2]
		
		var replyAddr string
		if len(params) == 4 {
			replyAddr = params[3]
		}
		
		var body string
		
		if bodyLen > 0 {
			body, err = stream.ReadBody(bodyLen)
			if err != nil {
				stream.WriteError(err); return
			}
		}
		
		server.exchange.Send(toAddr, replyAddr, timeout, body)
	}
	
	handleQuery := func(params []string) {
		if len(params) != 3 {
			stream.WriteError(os.NewError("query format: ? bodyLen timeout toAddr")); return
		}
	
		bodyLen, err := strconv.Atoi(params[0])
		if err != nil {
			stream.WriteError(os.NewError("invalid body length format")); return
		}
		
		timeout, err := strconv.Atoi64(params[1])
		if err != nil {
			stream.WriteError(os.NewError("invalid timeout format")); return
		}
		
		toAddr := params[2]
		
		body, err := stream.ReadBody(bodyLen)
		if err != nil {
			stream.WriteError(err); return
		}
		
		msg := server.exchange.Query(toAddr, timeout, body)
		
		err = stream.WriteMessage(msg)
		if err != nil {
			stream.WriteError(err); return
		}
	}

	for !stream.closed {
		command, err := stream.ReadCommand()
		if err != nil {
			stream.WriteError(err)
			break
		}
		
		switch command[0] {
		case readyCommandStr:
			handleReady(command[1:])
		case messageCommandStr:
			handleMessage(command[1:])
		case queryCommandStr:
			handleQuery(command[1:])
		case quitCommandStr:
			stream.Close()	
		default:
			stream.WriteError(os.NewError("invalid command"))
		}
	}
}
