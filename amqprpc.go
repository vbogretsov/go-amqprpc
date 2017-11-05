package amqprpc

import (
	"errors"
	"net/rpc"
	"strconv"
	"sync"

	"github.com/streadway/amqp"
	"github.com/vmihailenco/msgpack"
)

type amqpCodec struct {
	current amqp.Delivery
	message <-chan amqp.Delivery
	channel *amqp.Channel
}

type clientCodec struct {
	amqpCodec
	queueName  string
	routingKey string
}

// WriteRequest must be safe for concurrent use by multiple goroutines.
func (self *clientCodec) WriteRequest(req *rpc.Request, val interface{}) error {
	body, err := msgpack.Marshal(val)
	if err != nil {
		return err
	}

	publishing := amqp.Publishing{
		ReplyTo:       req.ServiceMethod,
		CorrelationId: self.queueName,
		MessageId:     strconv.FormatUint(req.Seq, 10),
		Body:          body,
	}

	return self.channel.Publish("", self.routingKey, false, false, publishing)
}

func (self *clientCodec) ReadResponseHeader(resp *rpc.Response) error {
	self.current = <-self.message

	if err := self.current.Headers.Validate(); err != nil {
		return errors.New("invalid header: " + err.Error())
	}

	if err, ok := self.current.Headers["error"]; ok {
		errMsg, ok := err.(string)
		if !ok {
			return errors.New("header not a string")
		}
		resp.Error = errMsg
	}

	var err error
	resp.Seq, err = strconv.ParseUint(self.current.MessageId, 10, 64)
	if err != nil {
		return err
	}

	resp.ServiceMethod = self.current.ReplyTo

	return nil
}

func (self *clientCodec) ReadResponseBody(val interface{}) error {
	if val == nil {
		return nil
	}

	return msgpack.Unmarshal(self.current.Body, val)
}

func (self *clientCodec) Close() error {
	return self.channel.Close()
}

func NewClientCodec(conn *amqp.Connection, name string) (rpc.ClientCodec, error) {
	channel, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	serverQueue, err := channel.QueueDeclare(name, false, false, false, false, nil)
	if err != nil {
		return nil, err
	}

	if serverQueue.Consumers == 0 {
		return nil, errors.New("no consumers in queue")
	}

	queue, err := channel.QueueDeclare("", false, true, false, false, nil)
	if err != nil {
		return nil, err
	}

	message, err := channel.Consume(queue.Name, "", true, false, false, false, nil)
	if err != nil {
		return nil, err
	}

	client := clientCodec{
		amqpCodec: amqpCodec{
			message: message,
			channel: channel,
		},
		queueName:  queue.Name,
		routingKey: name,
	}

	return &client, nil
}

type serverCodec struct {
	amqpCodec
	mutex    sync.RWMutex
	requests map[uint64]amqp.Delivery
}

func (self *serverCodec) ReadRequestHeader(req *rpc.Request) error {
	self.current = <-self.message

	if self.current.CorrelationId == "" {
		return errors.New("no routing key in delivery")
	}

	var err error
	req.Seq, err = strconv.ParseUint(self.current.MessageId, 10, 64)
	if err != nil {
		return err
	}

	self.mutex.Lock()
	self.requests[req.Seq] = self.current
	self.mutex.Unlock()

	req.ServiceMethod = self.current.ReplyTo
	return nil
}

func (self *serverCodec) ReadRequestBody(val interface{}) error {
	if val == nil {
		return nil
	}

	return msgpack.Unmarshal(self.current.Body, val)
}

// WriteResponse must be safe for concurrent use by multiple goroutines.
func (self *serverCodec) WriteResponse(resp *rpc.Response, val interface{}) error {
	self.mutex.Lock()
	delivery := self.requests[resp.Seq]
	delete(self.requests, resp.Seq)
	self.mutex.Unlock()

	body, err := msgpack.Marshal(val)
	if err != nil {
		return err
	}

	publishing := amqp.Publishing{
		ReplyTo:       resp.ServiceMethod,
		MessageId:     delivery.MessageId,
		CorrelationId: delivery.CorrelationId,
		Body:          body,
	}

	if resp.Error != "" {
		publishing.Headers = amqp.Table{"error": resp.Error}
	}

	err = self.channel.Publish(
		"",
		delivery.CorrelationId,
		false,
		false,
		publishing,
	)

	return err
}

func (self *serverCodec) Close() error {
	return self.channel.Close()
}

func NewServerCodec(conn *amqp.Connection, name string) (rpc.ServerCodec, error) {
	channel, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	queue, err := channel.QueueDeclare(name, false, false, false, false, nil)
	if err != nil {
		return nil, err
	}

	message, err := channel.Consume(queue.Name, "", true, false, false, false, nil)
	if err != nil {
		return nil, err
	}

	server := serverCodec{
		amqpCodec: amqpCodec{
			message: message,
			channel: channel,
		},
		requests: map[uint64]amqp.Delivery{},
	}

	return &server, nil
}
