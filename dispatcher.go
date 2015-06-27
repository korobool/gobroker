package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/m4rw3r/uuid"
	zmq "github.com/pebbe/zmq4"
	"sync"
	"time"
)

const (
	MaxWorkers   = 16
	PollInterval = 1000 * time.Millisecond
)

// type WorkerMsg struct {
// 	Value string
// }

type WorkerInfo struct {
	workerId   uint8
	workerType string
	tasks      []string
	kaFailed   int8
	kaLast     int64
}

type Task struct {
	chResult       chan string
	workerIdentity uint32
}

type Locks struct {
	workers *sync.RWMutex
}

type Dispatcher struct {
	zmqSocket *zmq.Socket
	zmqPoller *zmq.Poller
	locks     Locks
	//TODO: synchronize this map
	//use this datastructure https://github.com/streamrail/concurrent-map
	tasks        map[uuid.UUID]*Task
	outboundMsgs chan []string
	workers      map[uint32]*WorkerInfo
	methods      map[string][]uint32
	ids          map[string][MaxWorkers]bool
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
		zmqSocket:    zmqSocket,
		zmqPoller:    zmqPoller,
		locks:        Locks{workers: new(sync.RWMutex)},
		tasks:        make(map[uuid.UUID]*Task),
		outboundMsgs: make(chan []string),
		workers:      make(map[uint32]*WorkerInfo),
		methods:      make(map[string][]uint32),
		ids:          map[string][MaxWorkers]bool{},
	}, err
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
		d.ids[workerType] = [MaxWorkers]bool{}
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

		sockets, err := d.zmqPoller.Poll(PollInterval)
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

			// buf := new(bytes.Buffer)
			// buf.WriteByte(0x0)
			// err = binary.Write(buf, binary.LittleEndian, identity)

			strIdentity := identityIntToString(identity)

			d.zmqSocket.SendMessage(strIdentity, PROTO_KA, fmt.Sprintf("%d", d.workers[identity].workerId))
		}

		// TODO: We need to ignore all the messages from workers with unnkown identity

		if msg[1] == PROTO_KA {
			d.workers[identity].kaLast = time.Now().Unix()
			d.workers[identity].kaFailed = 0

			// TODO: this is done in one-client-mode just for protocol debugging. Implement it.
			time.Sleep(time.Second)

			// buf := new(bytes.Buffer)
			// buf.WriteByte(0x0)
			// err = binary.Write(buf, binary.LittleEndian, identity)

			strIdentity := identityIntToString(identity)

			if err != nil {
				fmt.Println("Error while prapring message")
			}
			// TODO: should store workerId as a string
			d.zmqSocket.SendMessage(strIdentity, PROTO_KA, fmt.Sprintf("%d", d.workers[identity].workerId))
		}
		if msg[1] == PROTO_DONE {

			if len(msg) < 3 {
				return errors.New("Malformed message")
			}
			taskId, err := uuid.FromString(msg[2])
			if err != nil {
				return errors.New("Wrong UUID in message")
			}

			if task, ok := d.tasks[taskId]; ok {
				fmt.Println("IN DONE!!!!!!")
				task.chResult <- msg[3]
			}

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

func (d *Dispatcher) ZmqWriteLoopRun() {
	for {
		select {
		case msg := <-d.outboundMsgs:
			d.zmqSocket.SendMessage(msg)
		case <-time.After(time.Second * 2):
			fmt.Println("timeout")
		}
	}
}

func (d *Dispatcher) getBestWorker(methodName string) uint32 {

	fmt.Println("!!!!!!!!!!!!!!!!", d, GrossDispatcher)
	candidates := d.methods[methodName]
	shortest := ^int(0)
	idx := 0

	// TODO: check for safity and put a Mutex if needed.
	for index, candidate := range candidates {
		if len(d.workers[candidate].tasks) < shortest {
			shortest = len(d.workers[candidate].tasks)
			idx = index
		}
	}

	return candidates[idx]
}

func (d *Dispatcher) ExecuteMethod(msg *ApiMessage, chResponse chan string) {
	fmt.Println("ExecuteMethod:", msg)

	// TODO: validations and errors
	// Select a worker

	bestWorker := d.getBestWorker(msg.method)

	// Generate TaskID and chanel for response
	taskUUID, _ := uuid.V4()
	chResult := make(chan string)

	// add Task to d.tsasks
	// TODO: add check if _, ok := d.tasks[taskUuid]; !ok .......
	d.tasks[taskUUID] = &Task{
		chResult:       chResult,
		workerIdentity: bestWorker,
	}

	// Build and send message with type TASK using zmq writing loop
	// buf := new(bytes.Buffer)
	// buf.WriteByte(0x0)
	// err = binary.Write(buf, binary.LittleEndian, identity)

	strIdentity := identityIntToString(bestWorker)

	d.outboundMsgs <- []string{
		strIdentity,
		PROTO_TASK,
		taskUUID.String(),
		msg.method,
		msg.params,
	}

	// setup cahanel listener for response with timeout

	// TODO: Check chanels for existance etc.
	select {
	case response := <-chResult:
		fmt.Printf("ExecuteMethod got result %s", response)
		chResponse <- response
		fmt.Println("ExecuteMethod write response")

		// select {
		// 	case chResponse, ok <- response:
		// 		if !ok {
		// 			fmt.Println("Chanel closed")
		// 			 return
		// 		}
		// }
	case <-time.After(time.Second * 2):
		fmt.Println("timeout")
	}

}
