package main

import (
	"fmt"
	zmq "github.com/pebbe/zmq4"
	// "strings"
	//"bytes"
	"encoding/binary"
	"time"
)

const (
	POLL_INTERVAL = 1000 * time.Millisecond
)

type WorkerMsg struct {
	Value string
}

type Dispatcher struct {
	zmqSocket *zmq.Socket
	zmqPoller *zmq.Poller

	workerMsgs chan *WorkerMsg
}

func NewDispatcher(uri string) (*Dispatcher, error) {

	zmqSocket, err := zmq.NewSocket(zmq.ROUTER)
	if err != nil {
		return nil, err
	}

	if err := zmqSocket.Bind(uri); err != nil {
		return nil, err
	}

	zmqPoller := zmq.NewPoller()
	zmqPoller.Add(zmqSocket, zmq.POLLIN)

	return &Dispatcher{
		zmqSocket: zmqSocket,
		zmqPoller: zmqPoller,
	}, err
}

func (d *Dispatcher) ZmqWriteLoopRun() error {
	return nil
}

func (d *Dispatcher) ZmqReadLoopRun() error {
	for {

		sockets, err := d.zmqPoller.Poll(POLL_INTERVAL)
		if err != nil {
			fmt.Println(sockets)
			break //  Interrupted
		}

		if len(sockets) != 1 {
			continue
		}

		msg, err := d.zmqSocket.RecvMessage(0)

		if err != nil {
			break
			fmt.Println("ZmqReadLoopRun FAILED!")
		}

		//var identity uint32
		b := []byte(msg[0][1:])
		//buf := bytes.NewReader(b)
		//err = binary.Read(buf, binary.LittleEndian, &identity)
		identity, _ := binary.Uvarint(b)

		fmt.Printf("[id:%d] recieved: %q\n", identity, msg)

		// if len(msg) == 1 {
		// 	if msg[0] == PROTO_KA {
		// 		worker.SendMessage(PROTO_KA)
		// 	}
		// 	} else if len(msg) == 3 {
		// 		if msg[0] == PROTO_TASK {
		// 			ok := queue.Put(taskqueue.Task{msg[1], msg[2]})
		// 			if ok {
		// 				go func(ch chan<- struct{}) { ch <- struct{}{} }(ch_task)
		// 				worker.SendMessage(PROTO_ACK, msg[1], PROTO_TASK)
		// 			} else {
		// 				worker.SendMessage(PROTO_NACK, msg[1], PROTO_TASK)
		// 			}
		// 		}
		// 	} else if len(msg) == 2 {
		// 		if msg[0] == PROTO_CANCEL {
		// 			go func(ch chan<- string, msg string) { ch <- msg }(ch_cancel, msg[1])
		// 		}
		// 	}
	}
	return nil
}

func (d *Dispatcher) ExecuteMethod(msg *ApiMessage) {
	fmt.Println("ExecuteMethod:", msg)

	// Build a call message to send to worker
	msgVal := &WorkerMsg{
		Value: msg.params,
	}
	// Send the message
	select {
	case d.workerMsgs <- msgVal: // <<< msg should be replaced by already parsed data
	case <-time.After(time.Second * 2):
		fmt.Println("Dispatcher workerMsgs timeout")
	}
}
