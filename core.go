package msglite

import (
	"fmt"
	"rand"
	"container/vector"
)

type Message struct {
	ToAddress string
	ReplyAddress string
	Broadcast bool
	Body string
}

func NewMessage(toAddress string, replyAddress string, broadcast bool, body string) Message {
	return Message{toAddress, replyAddress, broadcast, body}
}

type readyState struct {
	onAddress string
	messageChan chan <- Message
}

type Exchange struct {
	readyStateChan       chan readyState
	messageChan          chan Message
	unusedAddressReqChan chan (chan string)
	
	readyMap             map [string] *vector.Vector
	unreadyQueues        map [string] *vector.Vector
}

func NewExchange() (exchange *Exchange) {
	exchange = &Exchange{
		make(chan readyState),
		make(chan Message),
		make(chan (chan string)),
		make(map [string] *vector.Vector),
		make(map [string] *vector.Vector),
	}

	go func() {
		for {
			select {
			case rs := <- exchange.readyStateChan:
				exchange.handleReadyState(rs)
			case m := <- exchange.messageChan:
				exchange.handleMessage(m)
			case replyChan := <- exchange.unusedAddressReqChan:
				exchange.handleUnusedAddressReq(replyChan)
			}
		}
	}()
	
	return
}

func (exchange *Exchange) handleReadyState(rs readyState) {
	if unreadyQueue := exchange.unreadyQueues[rs.onAddress]; unreadyQueue != nil && unreadyQueue.Len() > 0 {
		rs.messageChan <- unreadyQueue.At(0).(Message)
		unreadyQueue.Delete(0)
		if unreadyQueue.Len() == 0 {
			exchange.unreadyQueues[rs.onAddress] = nil
		}
	} else {
		if exchange.readyMap[rs.onAddress] == nil {
			exchange.readyMap[rs.onAddress] = new(vector.Vector)
		}
		exchange.readyMap[rs.onAddress].Push(&rs)
	}
}

func (exchange *Exchange) handleMessage(m Message) {
	if readyStates := exchange.readyMap[m.ToAddress]; readyStates != nil && readyStates.Len() > 0 {
		switch {
		case m.Broadcast:
			for i := 0; i < readyStates.Len(); i++ {
				readyStates.At(i).(*readyState).messageChan <- m
			}
			readyStates.Resize(0, readyStates.Cap())
		
		default:
			readyStates.At(0).(*readyState).messageChan <- m
			readyStates.Delete(0)
			if readyStates.Len() == 0 {
				exchange.readyMap[m.ToAddress] = nil
			}
		}
	} else {
		if exchange.unreadyQueues[m.ToAddress] == nil {
			exchange.unreadyQueues[m.ToAddress] = new(vector.Vector)
		}
		exchange.unreadyQueues[m.ToAddress].Push(m)
	}
}

func (exchange *Exchange) handleUnusedAddressReq(replyChan chan string) {
	for {
		randAddr := fmt.Sprintf("%v", rand.Float())
		if _, ok := exchange.readyMap[randAddr]; !ok {
			replyChan <- randAddr
			break
		}
	}
}

func (exchange *Exchange) GenerateUnusedAddress() string {
	replyAddrChan := make(chan string)
	exchange.unusedAddressReqChan <- replyAddrChan
	return <- replyAddrChan
}

func (exchange *Exchange) SendMessage(toAddress string, replyAddress string, broadcast bool, body string) {
	exchange.messageChan <- NewMessage(toAddress, replyAddress, broadcast, body)
}

func (exchange *Exchange) SendQuery(toAddress string, body string) Message {
	replyAddr := exchange.GenerateUnusedAddress()
	exchange.messageChan <- NewMessage(toAddress, replyAddr, false, body)
	return <- exchange.ReadyOnAddress(replyAddr)
}

func (exchange *Exchange) ReadyOnAddress(onAddress string) <-chan Message {
	messageChan := make(chan Message)
	exchange.readyStateChan <- readyState{onAddress, messageChan}
	return messageChan
}

