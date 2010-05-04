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
}

func NewExchange() (exchange *Exchange) {
	exchange = &Exchange{
		make(chan readyState),
		make(chan Message),
		make(chan (chan string)),
		make(map [string] *vector.Vector),
		make(map [string] *vector.Vector),
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

func (exchange *Exchange) handleReadyState(rs readyState) {
	fmt.Printf("cilent ready on '%v'", rs.onAddress)

	if messageQueue, exists := exchange.messageQueues[rs.onAddress]; exists {
		fmt.Printf(" <- %v queued messages, sending now\n", messageQueue.Len())
		
		rs.messageChan <- messageQueue.At(0).(Message)
		messageQueue.Delete(0)
		if messageQueue.Len() == 0 {
			exchange.messageQueues[rs.onAddress] = nil, false
		}
	} else {
		fmt.Printf(" <- 0 queued messages, waiting...\n")
		
		if exchange.readyStateQueues[rs.onAddress] == nil {
			exchange.readyStateQueues[rs.onAddress] = new(vector.Vector)
		}
		exchange.readyStateQueues[rs.onAddress].Push(&rs)
	}
}

func (exchange *Exchange) handleMessage(m Message) {
	fmt.Printf("message from '%v' to '%v'", m.ReplyAddress, m.ToAddress)
	
	if readyStateQueue, exists := exchange.readyStateQueues[m.ToAddress]; exists {
		fmt.Printf(" -> %v clients ready, sending now\n", readyStateQueue.Len())
	
		readyStateQueue.At(0).(*readyState).messageChan <- m
		readyStateQueue.Delete(0)
		if readyStateQueue.Len() == 0 {
			exchange.readyStateQueues[m.ToAddress] = nil, false
		}
	} else {
		fmt.Printf(" -> 0 clients ready, queueing...\n")
	
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
				fmt.Printf("send timeout for message to '%v' from '%v'\n", toAddress, msg.ReplyAddress)
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
				fmt.Printf("ready timeout for client on '%v'\n", onAddress)
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

