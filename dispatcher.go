package main

import (
	//"fmt"
	"error"
	zmq "github.com/pebbe/zmq4"
)

type Dispatcher struct {
	zmqScoket *zmq.Socket
	zmqPoller *zmq.Poller
}

func NewDispatcher(uri string) (*Dispatcher, error) {

	var dispatcher Dispatcher

	if zmqSocket, err := zmq.NewSocket(zmq.ROUTER); err != nil {
		return nil, err
	}
	if err = zmqSocket.Bind(uri); err != nil {
		return nil, err
	}
	zmqPoller := zmq.NewPoller()
	zmqPoller.Add(zmqSocket, zmq.POLLIN)

	dispacther.zmqScoket = zmqScoket
	dispacther.zmqPoller = zmqPoller

	return &dispacther, err
}
