package main

import (
	//"fmt"
	zmq "github.com/pebbe/zmq4"
)

func NewDispatcher(uri string) (*Dispatcher, error) {
	zmqSocket, _ := zmq.NewSocket(zmq.ROUTER)
}
