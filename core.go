// Copyright (c) 2010 William R. Conant, WillConant.com
// Use of this source code is governed by the MIT licence:
// http://www.opensource.org/licenses/mit-license.php

package msglite

import (
	"fmt"
	"container/vector"
	"time"
	"strings"
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

const maxOnAddresses = 8

type readyState struct {
	onAddresses [maxOnAddresses]string
	onAddressCount int
	timeout int64
	messageChan chan <- Message
	timeoutReceived bool
}

type Exchange struct {
	readyStateChan       chan *readyState
	messageChan          chan Message
	unusedAddressReqChan chan (chan string)
	
	readyStateQueues     map [string] *vector.Vector
	messageQueues        map [string] *vector.Vector
	
	logLevel             int
	unusedAddressCounter uint32
}

func NewExchange() (exchange *Exchange) {
	exchange = &Exchange{
		make(chan *readyState),
		make(chan Message),
		make(chan (chan string)),
		make(map [string] *vector.Vector),
		make(map [string] *vector.Vector),
		LogLevelInfo,
		0,
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

func (exchange *Exchange) handleReadyState(rs *readyState) {
	if exchange.logLevel == LogLevelDebug {
		exchange.logf(LogLevelDebug, "< _ %v", strings.Join(rs.onAddresses[0:rs.onAddressCount], " "))
	}
	
	for i := 0; i < rs.onAddressCount; i++ {
		if messageQueue, exists := exchange.messageQueues[rs.onAddresses[i]]; exists {
			m := messageQueue.At(0).(Message)
			
			exchange.logf(LogLevelInfo, "> %v %v %v %v", len(m.Body), m.TimeoutSeconds, m.ToAddress, m.ReplyAddress)
			exchange.logf(LogLevelInfo, "  received, %v left in queue", messageQueue.Len() - 1)
			
			rs.messageChan <- m
			messageQueue.Delete(0)
			if messageQueue.Len() == 0 {
				exchange.messageQueues[rs.onAddresses[i]] = nil, false
			}
			
			return
		}
	}
	
	// no queued messages
	exchange.logf(LogLevelDebug, "  waiting")
	
	for i := 0; i < rs.onAddressCount; i++ {
		if exchange.readyStateQueues[rs.onAddresses[i]] == nil {
			exchange.readyStateQueues[rs.onAddresses[i]] = new(vector.Vector)
		}
		exchange.readyStateQueues[rs.onAddresses[i]].Push(rs)
	}
}

func (exchange *Exchange) unqueueReadyState(rs *readyState) {
	for i := 0; i < rs.onAddressCount; i++ {
		readyStateQueue := exchange.readyStateQueues[rs.onAddresses[i]]
		for j := 0; j < readyStateQueue.Len(); j++ {
			if readyStateQueue.At(j).(*readyState) == rs {
				readyStateQueue.Delete(j)
				if readyStateQueue.Len() == 0 {
					exchange.readyStateQueues[rs.onAddresses[i]] = nil, false
				}
				break
			}
		}
	}
}

func (exchange *Exchange) handleMessage(m Message) {
	exchange.logf(LogLevelInfo, "> %v %v %v %v", len(m.Body), m.TimeoutSeconds, m.ToAddress, m.ReplyAddress)
	
	if readyStateQueue, exists := exchange.readyStateQueues[m.ToAddress]; exists {
		exchange.logf(LogLevelInfo, "  delivered")
		
		rs := readyStateQueue.At(0).(*readyState)
		rs.messageChan <- m
		exchange.unqueueReadyState(rs)		
	} else {
		exchange.logf(LogLevelInfo, "  queued")
	
		if exchange.messageQueues[m.ToAddress] == nil {
			exchange.messageQueues[m.ToAddress] = new(vector.Vector)
		}
		exchange.messageQueues[m.ToAddress].Push(m)
	}
}

func (exchange *Exchange) handleUnusedAddressReq(replyChan chan string) {
	exchange.unusedAddressCounter++
	replyChan <- fmt.Sprintf("%X.%X", time.Seconds(), exchange.unusedAddressCounter)
}

func (exchange *Exchange) handleTick(t int64) {
	removeTheseUnreadyQueues := new (vector.StringVector)
	for toAddress, messageQueue := range(exchange.messageQueues) {
		for i := 0; i < messageQueue.Len(); i++ {
			msg := messageQueue.At(i).(Message)
			if msg.timeout < t {
				exchange.logf(LogLevelDebug, "> %v %v %v %v", len(msg.Body), msg.TimeoutSeconds, msg.ToAddress, msg.ReplyAddress)
				exchange.logf(LogLevelDebug, "  send timeout")
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
				exchange.logf(LogLevelDebug, "* ready timeout %v", onAddress)
				if !readyState.timeoutReceived {
					readyState.messageChan <- Message{"", "", 0, "", 0}
					readyState.timeoutReceived = true
				}
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

func (exchange *Exchange) Send(body string, timeoutSeconds int64, toAddress string, replyAddress string) {
	exchange.messageChan <- Message{toAddress, replyAddress, timeoutSeconds, body, time.Nanoseconds() + (timeoutSeconds * 1e9)}
}

func (exchange *Exchange) Query(body string, timeoutSeconds int64, toAddress string) *Message {
	replyAddr := exchange.GenerateUnusedAddress()
	exchange.messageChan <- Message{toAddress, replyAddr, timeoutSeconds, body, time.Nanoseconds() + (timeoutSeconds * 1e9)}
	return exchange.Ready(timeoutSeconds, []string{replyAddr})
}

func (exchange *Exchange) Ready(timeoutSeconds int64, onAddresses []string) *Message {
	rs := new(readyState)
	for i := 0; i < len(onAddresses); i++ {
		rs.onAddresses[i] = onAddresses[i]
	}
	rs.onAddressCount = len(onAddresses)
	rs.timeout = time.Nanoseconds() + (timeoutSeconds * 1e9)
	messageChan := make(chan Message)
	rs.messageChan = messageChan

	exchange.readyStateChan <- rs
	replyMsg := <-messageChan
	if replyMsg.ToAddress == "" {
		return nil
	}
	return &replyMsg
}
