package main

import (
	"log"
	"math/rand"
	"net/rpc"
	"sync"
	"time"

	"github.com/streadway/amqp"
	"github.com/vbogretsov/go-amqprpc"
	"github.com/vmihailenco/msgpack"
)

type MsgPackCodec struct{}

func (c *MsgPackCodec) Marshal(val interface{}) ([]byte, error) {
	return msgpack.Marshal(val)
}

func (c *MsgPackCodec) Unmarshal(data []byte, val interface{}) error {
	return msgpack.Unmarshal(data, val)
}

type Args struct {
	A int
	B int
}

func main() {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Fatal(err)
	}

	clientCodec, err := amqprpc.NewClientCodec(conn, "testrpc")
	if err != nil {
		log.Fatal(err)
	}
	defer clientCodec.Close()

	client := rpc.NewClientWithCodec(clientCodec)

	numCalls := 10000
	wg := sync.WaitGroup{}
	wg.Add(numCalls)
	sem := make(chan int, 100)

	t0 := time.Now()
	for i := 0; i < numCalls; i++ {
		sem <- 1
		go func() {
			args := Args{rand.Int() % 100, rand.Int() % 100}
			var result int

			err = client.Call("Test.Mul", args, &result)
			if err != nil {
				log.Fatal(err)
			}
			if result != args.A*args.B {
				log.Printf("%v * %v != %v", args.A, args.B, result)
				log.Fatal("FAIL")
			}

			wg.Done()
			<-sem
		}()
	}
	wg.Wait()
	log.Printf("SUCCESS, rps: %v", float64(numCalls)/time.Now().Sub(t0).Seconds())
}
