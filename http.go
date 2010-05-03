package msglite

import (
	"bytes"
	"http"
)

type HttpServer struct{
	exchange *Exchange
	listenNetwork string
	listenAddress string
	exchangeToAddress string
}

func NewHttpServer(exchange *Exchange, listenNetwork string, listenAddress string, exchangeToAddress string) (server *HttpServer) {
	server = &HttpServer{exchange, listenNetwork, listenAddress, exchangeToAddress}
	return
}

func (server *HttpServer) Run() {
	handler := func(conn *http.Conn, req *http.Request) {
		var msgBuf bytes.Buffer
		
		msgBuf.WriteString(req.Method + "\n")
		msgBuf.WriteString(req.RawURL + "\n")
		msgBuf.WriteString(req.Proto + "\n")
		msgBuf.WriteString("Host: " + req.Host + "\n")
		msgBuf.WriteString("User-Agent: " + req.UserAgent + "\n")
		msgBuf.WriteString("Referer: " + req.Referer + "\n")
		
		for k, v := range(req.Header) {
			msgBuf.WriteString(k + ": " + v + "\n")
		}
		
		msgBuf.WriteString("\n")
		
		var bodyBuf bytes.Buffer
		var inBuf [4096]byte
		
		for {
			nr, err := req.Body.Read(&inBuf)
			
			if nr > 0 {
				bodyBuf.Write(inBuf[0:nr])
			}
			
			if err == os.EOF {
				break
			}
			
			if err != nil {
				// TODO: make this more graceful
				panic("error reading http request body")
			}
		}
		
		msgBuf.
	}
}