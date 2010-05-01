package main

import (
	"./msglite"
	"fmt"
	"time"
)

func main() {
	exchange := msglite.NewExchange()
		
	go func() {
		for {
			msg := <- exchange.ReadyOnAddress("/sphynx")
			fmt.Printf("- query received: %v\n- sending reply to %v\n", msg.Body, msg.ReplyAddress)
			exchange.SendMessage(msg.ReplyAddress, "", false, "The answer to your question is 9.")
		}
	}()
	
	ticker := time.NewTicker(1e9 / 2)
	
	for i := 0; i < 100; i++ {
		<- ticker.C
		fmt.Printf("+ questioning the sphynx\n")
		reply := exchange.SendQuery("/sphynx", "Why do birds sing?")
		fmt.Printf("+ answer received: %v\n\n", reply.Body)
		
		// get someone to listen on the broadcast
		go func(myI int) {
			broadcastMsg := <- exchange.ReadyOnAddress("/shouts")
			fmt.Printf("* %v %v\n", myI, broadcastMsg.Body)
		}(i)
		
		if i % 10 == 0 {
			exchange.SendMessage("/shouts", "", true, "HAAAAAY!")
		}
	}
}
