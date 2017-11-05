package amqprpc_test

import (
	"math/rand"
	"net/rpc"
	"sync"
	"testing"

	"github.com/streadway/amqp"
	"github.com/vbogretsov/amqprpc"
)

type Args struct {
	A int
	B int
}

type Test struct{}

func (t *Test) Mul(args *Args, reply *int) error {
	*reply = args.A * args.B
	return nil
}

func TestSendRecv(t *testing.T) {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	serverCodec, err := amqprpc.NewServerCodec(conn, "testrpc")
	if err != nil {
		t.Fatal(err)
	}
	defer serverCodec.Close()

	rpc.Register(&Test{})

	clientCodec, err := amqprpc.NewClientCodec(conn, "testrpc")
	if err != nil {
		t.Fatal(err)
	}
	defer clientCodec.Close()

	client := rpc.NewClientWithCodec(clientCodec)

	go rpc.ServeCodec(serverCodec)

	numCalls := 100
	wg := sync.WaitGroup{}
	wg.Add(numCalls)

	for i := 0; i < numCalls; i++ {
		go func() {
			args := Args{rand.Int() % 100, rand.Int() % 100}
			var result int

			err = client.Call("Test.Mul", args, &result)
			wg.Done()

			if err != nil {
				t.Fatal(err)
			}
			if result != args.A*args.B {
				t.Fatal("%v * %v != %v", args.A, args.B, result)
			}
		}()
	}

	wg.Wait()
}
