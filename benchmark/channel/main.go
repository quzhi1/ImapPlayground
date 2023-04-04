package main

import (
	"log"
	"time"
)

func main() {
	ch := make(chan int)

	go func() {
		select {
		case <-ch:
			log.Printf("1.channel")
		default:
			log.Printf("1.default")
		}
		<-ch
		log.Printf("2.channel")
		close(ch)
		// Close of a closed channel
		// close(ch)
		select {
		case <-ch:
			log.Printf("3.channel")
		default:
			log.Printf("3.default")
		}
	}()
	time.Sleep(time.Second)
	ch <- 1
	time.Sleep(time.Second)

	// Close receive-only channel
	// ch := make(<-chan int)
	// close(ch)
	// fmt.Println("ok")

	// Close send-only channel
	// c := make(chan<- int)
	// close(c)
	// fmt.Println("ok")
}
