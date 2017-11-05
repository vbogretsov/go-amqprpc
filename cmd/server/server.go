package main

import (
	"log"
	"net/rpc"

	"github.com/streadway/amqp"
	"github.com/vbogretsov/amqprpc"
	// "github.com/vibhavp/amqp-rpc"
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

type Test struct{}

func (t *Test) Mul(args *Args, reply *int) error {
	*reply = args.A * args.B
	return nil
}

func main() {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Fatal(err)
	}

	serverCodec, err := amqprpc.NewServerCodec(conn, "testrpc")
	if err != nil {
		log.Fatal(err)
	}

	rpc.Register(&Test{})
	rpc.ServeCodec(serverCodec)
}
