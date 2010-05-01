package msglite

import (
	"net"
	"fmt"
)

type SocketServer struct {
	exchange Exchange
	listener net.Listener
}

func NewSocketServer(exchange Exchange, path string) (server *SocketServer) {
	server = new(SocketServer)
	server.exchange = exchange
	
	var err os.Error
	
	server.listener, err = net.Listen("unix", path)
	if err != nil {
		panic(fmt.Sprintf("couldn't listen at %v: %v", path, err))
	}
	
	
}