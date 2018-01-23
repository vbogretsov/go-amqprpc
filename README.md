# go-amqprpc

Go net/rpc codec implementation for AMQP.
Small rework of [amqprpc](https://github.com/vibhavp/amqp-rpc).

Updates:
* Fixed memory leak (server requests was not removed from map after processing)
* requests map replaced to sync.Map

*NOTE*: current implementation does not allow to provide custom encoder, msgpack is faster than gob and json and does not rquire code generation.

## Usage

### Server

```go
package main

import (
    "log"
    "net/rpc"

    "github.com/streadway/amqp"
    "github.com/vbogretsov/go-amqprpc"
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
    conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
    if err != nil {
        log.Fatal(err)
    }

    serverCodec, err := amqprpc.NewServerCodec(conn, "testrpc", amqprpc.MsgPack)
    if err != nil {
        log.Fatal(err)
    }

    rpc.Register(&Test{})
    rpc.ServeCodec(serverCodec)
}
```

### Client

```go
package main

import (
    "log"
    "math/rand"
    "net/rpc"
    "sync"
    "time"

    "github.com/streadway/amqp"
    "github.com/vbogretsov/go-amqprpc"
)

type Args struct {
    A int
    B int
}

func main() {
    conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
    if err != nil {
        log.Fatal(err)
    }

    clientCodec, err := amqprpc.NewClientCodec(conn, "testrpc", amqprpc.MsgPack)
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

            if err := client.Call("Test.Mul", args, &result); err != nil {
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
```

## Licence

See the LICENSE file.