# go-amqprpc

Go net/rpc codec implementation for AMQP.
Small rework of [amqprpc](https://github.com/vibhavp/amqp-rpc).

Updates:
* Fixed memory leak (server requests was not removed from map after processing)
* requests map replaced to sync.Map

*NOTE*: current implementation does not allow to provide custom encoder, msgpack is faster than gob and json and does not rquire code generation.

# Licence

See the LICENSE file.