package msglite

import (
	"fmt"
	"rand"
	"container/vector"
	"time"
)

const (
	AddressReadyTimeout = "msglite.ReadyTimeout"
)

const (
	_ = iota
	LogLevelMinimal
	LogLevelInfo
	LogLevelDebug
)

type Message struct {
	ToAddress string
	ReplyAddress string
	TimeoutSeconds int64
	Body string
	timeout int64
}

type readyState struct {
	onAddress string
	timeout int64
	messageChan chan <- Message
}

type Exchange struct {
	readyStateChan       chan readyState
	messageChan          chan Message
	unusedAddressReqChan chan (chan string)
	
	readyStateQueues     map [string] *vector.Vector
	messageQueues        map [string] *vector.Vector
	
	logLevel             int
}

func NewExchange() (exchange *Exchange) {
	exchange = &Exchange{
		make(chan readyState),
		make(chan Message),
		make(chan (chan string)),
		make(map [string] *vector.Vector),
		make(map [string] *vector.Vector),
		LogLevelInfo,
	}
	
	ticker := time.NewTicker(1e9)
	
	go func() {
		for {
			select {
			case rs := <-exchange.readyStateChan:
				exchange.handleReadyState(rs)
			case m := <-exchange.messageChan:
				exchange.handleMessage(m)
			case replyChan := <-exchange.unusedAddressReqChan:
				exchange.handleUnusedAddressReq(replyChan)
			case t := <-ticker.C:
				exchange.handleTick(t)
			}
		}
	}()
	
	return
}

func (exchange *Exchange) SetLogLevel(l int) {
	exchange.logLevel = l
}

func (exchange *Exchange) log(level int, s string) {
	if level <= exchange.logLevel {
		fmt.Println(s)
	}
}

func (exchange *Exchange) logf(level int, format string, v ...interface{}) {
	exchange.log(level, fmt.Sprintf(format, v))
}

func (exchange *Exchange) handleReadyState(rs readyState) {
	exchange.logf(LogLevelDebug, "! ready %v", rs.onAddress)

	if messageQueue, exists := exchange.messageQueues[rs.onAddress]; exists {
		m := messageQueue.At(0).(Message)
		
		exchange.logf(LogLevelInfo, "< %v (%v)", m.ToAddress, m.ReplyAddress)
		exchange.logf(LogLevelInfo, "  received, %v left in queue", messageQueue.Len() - 1)
		
		rs.messageChan <- m
		messageQueue.Delete(0)
		if messageQueue.Len() == 0 {
			exchange.messageQueues[rs.onAddress] = nil, false
		}
	} else {
		exchange.logf(LogLevelDebug, "  waiting")
		
		if exchange.readyStateQueues[rs.onAddress] == nil {
			exchange.readyStateQueues[rs.onAddress] = new(vector.Vector)
		}
		exchange.readyStateQueues[rs.onAddress].Push(&rs)
	}
}

func (exchange *Exchange) handleMessage(m Message) {
	exchange.logf(LogLevelInfo, "> %v (%v)", m.ToAddress, m.ReplyAddress)
	
	if readyStateQueue, exists := exchange.readyStateQueues[m.ToAddress]; exists {
		exchange.logf(LogLevelInfo, "  delivered")
	
		readyStateQueue.At(0).(*readyState).messageChan <- m
		readyStateQueue.Delete(0)
		if readyStateQueue.Len() == 0 {
			exchange.readyStateQueues[m.ToAddress] = nil, false
		}
	} else {
		exchange.logf(LogLevelDebug, "  queued")
	
		if exchange.messageQueues[m.ToAddress] == nil {
			exchange.messageQueues[m.ToAddress] = new(vector.Vector)
		}
		exchange.messageQueues[m.ToAddress].Push(m)
	}
}

func (exchange *Exchange) handleUnusedAddressReq(replyChan chan string) {
	for {
		randAddr := fmt.Sprintf("%v", rand.Float())
		if _, ok := exchange.readyStateQueues[randAddr]; !ok {
			replyChan <- randAddr
			break
		}
	}
}

func (exchange *Exchange) handleTick(t int64) {
	removeTheseUnreadyQueues := new (vector.StringVector)
	for toAddress, messageQueue := range(exchange.messageQueues) {
		for i := 0; i < messageQueue.Len(); i++ {
			msg := messageQueue.At(i).(Message)
			if msg.timeout < t {
				exchange.logf(LogLevelDebug, "! send timeout %v (%v)", toAddress, msg.ReplyAddress)
				messageQueue.Delete(i)
				i--
			}
		}
		if messageQueue.Len() == 0 {
			removeTheseUnreadyQueues.Push(toAddress)
		}
	}
	for i := 0; i < removeTheseUnreadyQueues.Len(); i++ {
		exchange.messageQueues[removeTheseUnreadyQueues.At(i)] = nil, false
	}

	removeTheseReadyStates := new(vector.StringVector)
	for onAddress, readyStateQueue := range(exchange.readyStateQueues) {
		for i := 0; i < readyStateQueue.Len(); i++ {
			readyState := readyStateQueue.At(i).(*readyState)
			if readyState.timeout < t {
				exchange.logf(LogLevelDebug, "! ready timeout %v", onAddress)
				readyState.messageChan <- Message{AddressReadyTimeout, "", 0, "", 0}
				readyStateQueue.Delete(i)
				i--
			}
		}
		if readyStateQueue.Len() == 0 {
			removeTheseReadyStates.Push(onAddress)
		}
	}
	for i := 0; i < removeTheseReadyStates.Len(); i++ {
		exchange.readyStateQueues[removeTheseReadyStates.At(i)] = nil, false
	}
}

func (exchange *Exchange) GenerateUnusedAddress() string {
	replyAddrChan := make(chan string)
	exchange.unusedAddressReqChan <- replyAddrChan
	return <- replyAddrChan
}

func (exchange *Exchange) Send(toAddress string, replyAddress string, timeoutSeconds int64, body string) {
	exchange.messageChan <- Message{toAddress, replyAddress, timeoutSeconds, body, time.Nanoseconds() + (timeoutSeconds * 1e9)}
}

func (exchange *Exchange) Query(toAddress string, timeoutSeconds int64, body string) Message {
	replyAddr := exchange.GenerateUnusedAddress()
	exchange.messageChan <- Message{toAddress, replyAddr, timeoutSeconds, body, time.Nanoseconds() + (timeoutSeconds * 1e9)}
	return <- exchange.Ready(replyAddr, timeoutSeconds)
}

func (exchange *Exchange) Ready(onAddress string, timeoutSeconds int64) <-chan Message {
	messageChan := make(chan Message)
	exchange.readyStateChan <- readyState{onAddress, time.Nanoseconds() + (timeoutSeconds * 1e9), messageChan}
	return messageChan
}

