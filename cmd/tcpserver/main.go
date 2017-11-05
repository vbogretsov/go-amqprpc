package main

import (
	"log"
	"net"
	"net/rpc"
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

func main() {
	addy, err := net.ResolveTCPAddr("tcp", "0.0.0.0:42586")
	if err != nil {
		log.Fatal(err)
	}

	inbound, err := net.ListenTCP("tcp", addy)
	if err != nil {
		log.Fatal(err)
	}

	rpc.Register(&Test{})
	rpc.Accept(inbound)
}
