package msglite

import (
	"net"
	"bytes"
	"http"
	"io/ioutil"
	"strings"
	"strconv"
	"bufio"
	"os"
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
		server.handleReq(conn, req)
	}
	
	listener, err := net.Listen(server.listenNetwork, server.listenAddress)
	if err != nil {
		panic(err)
	}
	
	http.Serve(listener, http.HandlerFunc(handler))
}

func (server *HttpServer) handleReq(conn *http.Conn, req *http.Request) {
	msgBody, err := server.transformReq(req)
	if err != nil {
		http.Error(conn, err.String(), http.StatusInternalServerError)
	}
	
	reply := server.exchange.Query(server.exchangeToAddress, 10, msgBody)
	
	server.transformReply(reply, conn)
}

func (server *HttpServer) transformReq(req *http.Request) (string, os.Error) {
	var msgBuf bytes.Buffer
	
	msgBuf.WriteString(req.Method + " " + req.RawURL + " " + req.Proto + "\n")
	msgBuf.WriteString("Host: " + req.Host + "\n")
	msgBuf.WriteString("User-Agent: " + req.UserAgent + "\n")
	msgBuf.WriteString("Referer: " + req.Referer + "\n")
	
	for k, v := range(req.Header) {
		msgBuf.WriteString(k + ": " + v + "\n")
	}
	
	msgBuf.WriteString("\n")
	
	bodySlice, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return "", err
	}
	
	msgBuf.Write(bodySlice)
	
	return msgBuf.String(), nil
}

func (server *HttpServer) transformReply(msg Message, conn *http.Conn) {
	reader := bufio.NewReader(strings.NewReader(msg.Body))
	
	statusLine, err := reader.ReadString('\n')
	if err != nil {
		goto BadResponse
	}
	
	status, err := strconv.Atoi(statusLine[0:len(statusLine)-1])
	if err != nil {
		goto BadResponse
	}
	
	for {
		headerLine, err := reader.ReadString('\n')
		if err != nil {
			goto BadResponse
		}
		
		if headerLine == "\n" {
			break
		}
		
		sepIdx := strings.Index(headerLine, ": ")
		header := headerLine[0:sepIdx]
		value := headerLine[sepIdx+2:len(headerLine)-1]

		conn.SetHeader(header, value)
	}
	
	bodySlice, _ := ioutil.ReadAll(reader)
	
	conn.WriteHeader(status)
	conn.Write(bodySlice)
	
	return
	
BadResponse:
	http.Error(conn, "bad response from upstream handler", http.StatusInternalServerError)
}
