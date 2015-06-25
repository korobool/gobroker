package main

import (
	//"fmt"
	"errors"
	zmq "github.com/pebbe/zmq4"
	"time"
)

const (
	POLL_INTERVAL = 1000 * time.Millisecond
)

type Dispatcher struct {
	zmqScoket *zmq.Socket
	zmqPoller *zmq.Poller
}

func NewDispatcher(uri string) (*Dispatcher, error) {

	if zmqSocket, err := zmq.NewSocket(zmq.ROUTER); err != nil {
		return nil, err
	}
	if err = zmqSocket.Bind(uri); err != nil {
		return nil, err
	}
	zmqPoller := zmq.NewPoller()
	zmqPoller.Add(zmqSocket, zmq.POLLIN)

	dispatcher := &Dispatcher{
		zmqSocket: zmqSocket,
		zmqPoller: zmqPoller,
	}
	return dispacther, err
}

func (d *Dispatcher) ZmqLoopRun() error {
	for {
		sockets, err := d.zmqPoller.Poll(POLL_INTERVAL)
		if err != nil {
			break //  Interrupted
		}
	}
}
