package main

import (
	"log"
	"math/rand"
	"net/rpc"
	"sync"
	"time"
)

type Args struct {
	A int
	B int
}

func main() {
	client, err := rpc.Dial("tcp", "localhost:42586")
	if err != nil {
		log.Fatal(err)
	}

	numCalls := 10000
	wg := sync.WaitGroup{}
	wg.Add(numCalls)

	t0 := time.Now()
	for i := 0; i < numCalls; i++ {
		go func() {
			var reply int
			args := Args{rand.Int() % 100, rand.Int() % 100}
			err = client.Call("Test.Mul", args, &reply)
			wg.Done()
			if err != nil {
				log.Fatal(err)
			}
			if reply != args.A*args.B {
				log.Fatal("invalid result")
			}
		}()
	}
	wg.Wait()

	log.Printf("SUCCESS rps: %v", float64(numCalls)/time.Now().Sub(t0).Seconds())
}
