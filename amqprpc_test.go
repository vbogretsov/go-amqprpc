package amqprpc_test

import (
	"flag"
	"math/rand"
	"net/rpc"
	"sync"
	"testing"

	"github.com/streadway/amqp"
	"github.com/vbogretsov/go-amqprpc"
)

var (
	amqpurl string
	nserver int
	nclient int
	ncalls  int
)

func init() {
	flag.StringVar(
		&amqpurl,
		"amqpurl",
		"amqp://guest:guest@localhost:5672/",
		"AMQP broker URL",
	)
	flag.IntVar(&nserver, "nserver", 1, "number of test servers")
	flag.IntVar(&nclient, "nclient", 1, "number of test clients")
	flag.IntVar(&ncalls, "ncalls", 1000, "number of server calls per client")
	flag.Parse()
}

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
	conn, err := amqp.Dial(amqpurl)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	serverCodecs := make([]rpc.ServerCodec, nserver)
	for i := 0; i < nserver; i++ {
		codec, err := amqprpc.NewServerCodec(conn, "testrpc", amqprpc.MsgPack)
		if err != nil {
			t.Fatal(err)
		}
		defer codec.Close()
		serverCodecs[i] = codec
	}

	clientCodecs := make([]rpc.ClientCodec, nclient)
	clients := make([]*rpc.Client, nclient)
	for i := 0; i < nclient; i++ {
		codec, err := amqprpc.NewClientCodec(conn, "testrpc", amqprpc.MsgPack)
		if err != nil {
			t.Fatal(err)
		}
		defer codec.Close()
		clientCodecs[i] = codec

		client := rpc.NewClientWithCodec(codec)
		defer client.Close()
		clients[i] = client
	}

	rpc.Register(&Test{})

	for i := 0; i < nserver; i++ {
		go rpc.ServeCodec(serverCodecs[i])
	}

	wg := sync.WaitGroup{}
	wg.Add(ncalls * nclient)

	for i := 0; i < ncalls; i++ {
		for j := 0; j < nclient; j++ {
			go func(cli int) {
				args := Args{rand.Int() % 100, rand.Int() % 100}
				var result int

				if err := clients[cli].Call("Test.Mul", args, &result); err != nil {
					t.Fatal(err)
				}

				wg.Done()

				if result != args.A*args.B {
					t.Fatal("%v * %v != %v", args.A, args.B, result)
				}
			}(j)
		}
	}

	wg.Wait()
}
