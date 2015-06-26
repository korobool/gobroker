package main

import (
	"fmt"
	zmq "github.com/pebbe/zmq4"
	// "strings"
	"bytes"
	"encoding/binary"
	// "strconv"
	"sync"
	"time"
)

const (
	POLL_INTERVAL = 1000 * time.Millisecond
)

type WorkerMsg struct {
	Value string
}

type WorkerInfo struct {
	workerId   uint8
	workerType string
	tasks      []string
	kaFailed   int8
	kaLast     int64
}

type Locks struct {
	workers *sync.RWMutex
}

type Dispatcher struct {
	zmqSocket  *zmq.Socket
	zmqPoller  *zmq.Poller
	locks      Locks
	workerMsgs chan *WorkerMsg
	workers    map[uint32]*WorkerInfo
	methods    map[string][]uint32
	ids        map[string][16]bool
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
		zmqSocket:  zmqSocket,
		zmqPoller:  zmqPoller,
		locks:      Locks{workers: new(sync.RWMutex)},
		workerMsgs: make(chan *WorkerMsg),
		workers:    make(map[uint32]*WorkerInfo),
		methods:    make(map[string][]uint32),
		ids:        map[string][16]bool{},
	}, err
}

func (d *Dispatcher) ZmqWriteLoopRun() error {
	return nil
}

func (d *Dispatcher) addWorker(identity uint32, msg []string) error {

	d.locks.workers.Lock()
	defer d.locks.workers.Unlock()

	workerType := msg[2]

	workerId := d.takeWorkerId(workerType)

	d.workers[identity] = &WorkerInfo{
		workerId:   workerId,
		workerType: workerType,
		tasks:      []string{},
		kaLast:     0,
		kaFailed:   0,
	}

	// Get methods exposed by a worker
	exposedMethods := msg[3:]

	for idx := range exposedMethods {
		method := exposedMethods[idx]
		if _, ok := d.methods[method]; ok {
			d.methods[method] = append(d.methods[method], identity)
		} else {
			d.methods[method] = []uint32{identity}
		}

	}
	return nil
}

func (d *Dispatcher) removeWorker(identity uint32) {
	d.locks.workers.Lock()
	defer d.locks.workers.Unlock()

}

func (d *Dispatcher) releaseWorkerId(id uint8, workerType string) {
	ids := d.ids[workerType]
	ids[id-1] = false
	d.ids[workerType] = ids
}

func (d *Dispatcher) takeWorkerId(workerType string) uint8 {

	// TODO: Take id lists from the map
	if _, ok := d.ids[workerType]; !ok {
		d.ids[workerType] = [16]bool{}
	}

	for idx, val := range d.ids[workerType] {

		if !val {
			ids := d.ids[workerType]
			ids[idx] = true
			d.ids[workerType] = ids
			return uint8(idx + 1)
		}
	}

	return 0
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

		// Ugly 4 bytes to int32 conversion (msg[0][1:] has length 4)
		identity := binary.LittleEndian.Uint32([]byte(msg[0][1:]))

		fmt.Printf("[id:%d] recieved: %q\n", identity, msg)
		//fmt.Printf("!!!!!!!!!!!!!", d.ids)

		cmd := msg[1]
		if cmd == PROTO_READY {
			if _, ok := d.workers[identity]; ok {
				d.removeWorker(identity)
			}

			d.addWorker(identity, msg)

			//buf := make([]byte, 8)

			buf := new(bytes.Buffer)
			buf.Write(0x0)
			err = binary.Write(buf, binary.LittleEndian, identity)

			//binary.PutUvarint(buf, uint64(identity))
			//buf = append([]byte{0x0}, buf...)

			fmt.Println(buf.Bytes())

			d.zmqSocket.SendMessage(string(buf.Bytes()), PROTO_KA)

		}

		// TODO: We need to ignore all the messages from workers with unnkown identity

		if msg[1] == PROTO_KA {
			d.workers[identity].kaLast = time.Now().Unix()
			d.workers[identity].kaFailed = 0

			// TODO: this is done in one-client-mode just for protocol debugging. Implement it.
			time.Sleep(time.Second)

			var buf []byte
			binary.PutUvarint(buf, uint64(identity))
			buf = append([]byte{0x0}, buf...)

			fmt.Println(string(buf))

			d.zmqSocket.SendMessage(PROTO_KA, string(buf))
		}

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

func remoteCall(apiMsg *ApiMessage, chResult chan string) {

	go dispatcher.ExecuteMethod(apiMsg)

	select {
	case chResult <- strings.ToUpper(apiMsg.params):

	case <-time.After(time.Second * 2):
		fmt.Println("timeout")
	}
}
