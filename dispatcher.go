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
	MaxWorkers           = 16
	PollInterval         = 10 * time.Millisecond
	AliveTimeout         = 2
	MaxKAFailed          = 2
	HeartbeatingInterval = 1 * time.Second
	WorkerMaxTasks       = 100
)

// type WorkerMsg struct {
// 	Value string
// }

type WorkerInfo struct {
	workerId   uint8
	workerType string
	tasks      []TaskId
	kaFailed   int8
	kaLast     int64
}

type TaskId struct {
	workerIdentity uint32
	taskUUID       uuid.UUID
}

type Task struct {
	chResult       chan string
	workerIdentity uint32
}

type Locks struct {
	workers *sync.RWMutex
	socket  *sync.Mutex
	tasks   *sync.Mutex
}

type Dispatcher struct {
	zmqSocket *zmq.Socket
	zmqPoller *zmq.Poller
	locks     Locks
	//TODO: synchronize this map
	//use this datastructure https://github.com/streamrail/concurrent-map
	tasks        map[TaskId]*Task
	outboundMsgs chan []string
	workers      map[uint32]*WorkerInfo
	methods      map[string][]uint32
	ids          map[string][MaxWorkers]bool
}

func NewDispatcher(uri string) (*Dispatcher, error) {
	//context, err := zmq.NewContext()
	//if err != nil {
	//	return nil, err
	//}
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
		locks:        Locks{workers: new(sync.RWMutex), socket: new(sync.Mutex), tasks: new(sync.Mutex)},
		tasks:        make(map[TaskId]*Task),
		outboundMsgs: make(chan []string, 100),
		workers:      make(map[uint32]*WorkerInfo),
		methods:      make(map[string][]uint32),
		ids:          map[string][MaxWorkers]bool{},
	}, err
}

func (d *Dispatcher) run() {
	// Starting zeromq loop
	go d.ZmqReadLoopRun()
	//go d.ZmqWriteLoopRun()
	go d.HeartbeatingRun()
}

func (d *Dispatcher) HeartbeatingRun() {
	for {

		removeList := []uint32{}

		now := time.Now().Unix()
		for identity, worker := range d.workers {
			if now-worker.kaLast > AliveTimeout {
				if worker.kaFailed > MaxKAFailed {
					removeList = append(removeList, identity)
					continue
				} else {
					d.workers[identity].kaFailed += 1
				}
			}
			strIdentity := identityIntToString(identity)
			//d.outboundMsgs <- []string{strIdentity, PROTO_KA, fmt.Sprintf("%d", worker.workerId)}
			kaMsg := []string{strIdentity, PROTO_KA, fmt.Sprintf("%d", worker.workerId)}
			go d.sendToOutbound(kaMsg)

		}
		for _, id := range removeList {
			d.removeWorker(id)
		}
		time.Sleep(HeartbeatingInterval)
	}
}

func (d *Dispatcher) addWorker(identity uint32, msg []string) error {

	d.locks.workers.Lock()
	defer d.locks.workers.Unlock()

	workerType := msg[2]

	workerId, err := d.takeWorkerId(workerType)
	if err != nil {
		return err
	}

	d.workers[identity] = &WorkerInfo{
		workerId:   workerId,
		workerType: workerType,
		tasks:      []TaskId{},
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

func (d *Dispatcher) removeTasks(taskIds []TaskId) {
	d.locks.tasks.Lock()
	defer d.locks.tasks.Unlock()

	for _, taskId := range taskIds {
		//close(d.tasks[taskId].chResult)
		delete(d.tasks, taskId)
	}
}

func (d *Dispatcher) removeWorker(identity uint32) {
	d.locks.workers.Lock()
	defer d.locks.workers.Unlock()

	for method, identitySlice := range d.methods {
		freshIdentitySlice := []uint32{}
		for _, id := range identitySlice {
			if id != identity {
				freshIdentitySlice = append(freshIdentitySlice, id)
			}
		}
		d.methods[method] = freshIdentitySlice
	}

	d.releaseWorkerId(d.workers[identity].workerId, d.workers[identity].workerType)

	d.removeTasks(d.workers[identity].tasks)

	delete(d.workers, identity)
}

func (d *Dispatcher) releaseWorkerId(id uint8, workerType string) {
	ids := d.ids[workerType]
	ids[id-1] = false
	d.ids[workerType] = ids
}

func (d *Dispatcher) takeWorkerId(workerType string) (uint8, error) {

	// TODO: Take id lists from the map
	if _, ok := d.ids[workerType]; !ok {
		d.ids[workerType] = [MaxWorkers]bool{}
	}

	for idx, val := range d.ids[workerType] {

		if !val {
			ids := d.ids[workerType]
			ids[idx] = true
			d.ids[workerType] = ids
			return uint8(idx + 1), nil
		}
	}

	return 0, errors.New("No free woker id")
}

func (d *Dispatcher) sendToOutbound(msg []string) {
	select {
	case d.outboundMsgs <- msg:
		fmt.Println(">>>sendToOutbound:", msg)
	case <-time.After(time.Second * 5):
		fmt.Println("sendToOutbound: failed by timeout")
	}
}

func (d *Dispatcher) ZmqReadLoopRun() error {
	for {
		for exitSelect := false; !exitSelect; {
			select {
			case msg := <-d.outboundMsgs:
				fmt.Println("<<<ZmqLoop: SENDING", msg)
				d.zmqSocket.SendMessage(msg)
				fmt.Println(">>>>ZmqLoop: SENT", msg)
			default:
				//fmt.Println(">>>>SEND default")
				exitSelect = true
			}
		}

		sockets, err := d.zmqPoller.Poll(PollInterval)
		if err != nil {
			fmt.Println(err)
			break //  Interrupted
		}
		if len(sockets) != 1 {
			continue
		}

		//d.locks.socket.Lock()
		msg, err := d.zmqSocket.RecvMessage(0)
		//d.locks.socket.Unlock()
		if err != nil {
			fmt.Println("ZmqReadLoopRun: RecvMessage FAILED!")
			break
		}

		// Ugly 4 bytes to int32 conversion (msg[0][1:] has length 4)
		identity := binary.LittleEndian.Uint32([]byte(msg[0][1:]))

		//fmt.Printf("[id:%d] recieved: %q\n", identity, msg)

		cmd := msg[1]
		if cmd == PROTO_READY {
			if _, ok := d.workers[identity]; ok {
				d.removeWorker(identity)
			}

			err = d.addWorker(identity, msg)
			if err != nil {
				fmt.Println(err)
				continue
			}

			strIdentity := identityIntToString(identity)

			firstValuebleKA := []string{strIdentity, PROTO_KA, fmt.Sprintf("%d", d.workers[identity].workerId)}
			//d.outboundMsgs <- firstValuebleKA
			go d.sendToOutbound(firstValuebleKA)
		}

		// TODO: We need to ignore all the messages from workers with unnkown identity
		if _, ok := d.workers[identity]; !ok {
			fmt.Println(">>>>Unknown identity", msg)
			continue
		}

		if msg[1] == PROTO_KA {
			d.workers[identity].kaLast = time.Now().Unix()
			d.workers[identity].kaFailed = 0

			if err != nil {
				fmt.Println("Error while prapring message")
			}
		}
		if msg[1] == PROTO_DONE {
			if len(msg) < 3 {
				return errors.New("Malformed message")
			}
			taskUUID, err := uuid.FromString(msg[2])
			if err != nil {
				return errors.New("Wrong UUID in message")
			}

			d.locks.tasks.Lock()
			if task, ok := d.tasks[TaskId{identity, taskUUID}]; ok {
				if task.chResult == nil {
					return errors.New("result channel closed")
				}
				task.chResult <- msg[3]
			}
			d.locks.tasks.Unlock()

		}
	}
	return nil
}

func (d *Dispatcher) ZmqWriteLoopRun() {
	for {
		select {
		case msg := <-d.outboundMsgs:
			d.locks.socket.Lock()
			d.zmqSocket.SendMessage(msg)
			d.locks.socket.Unlock()
		case <-time.After(time.Second * 10):
			fmt.Println("ZmqWriteLoopRun: timeout")
		}
	}
}

func (d *Dispatcher) getBestWorker(methodName string) (uint32, error) {

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

	if len(candidates) == 0 || len(candidates) > WorkerMaxTasks {
		return 0, errors.New("No free workers avaliable")
	}
	return candidates[idx], nil
}

func (d *Dispatcher) ExecuteMethod(msg *ApiMessage, chResponse chan string) {
	//fmt.Println("ExecuteMethod:", msg)

	// TODO: validations and errors
	// Select a worker

	bestWorker, err := d.getBestWorker(msg.method)
	if err != nil {
		close(chResponse)
		fmt.Println(err)
		return
	}

	// Generate TaskID and chanel for response
	taskUUID, _ := uuid.V4()
	chResult := make(chan string)

	taskId := TaskId{bestWorker, taskUUID}
	// add Task to d.tsasks
	// TODO: add check if _, ok := d.tasks[taskUuid]; !ok .......

	d.locks.tasks.Lock()
	d.tasks[taskId] = &Task{
		chResult:       chResult,
		workerIdentity: bestWorker,
	}
	d.workers[bestWorker].tasks = append(d.workers[bestWorker].tasks, taskId)
	d.locks.tasks.Unlock()

	strIdentity := identityIntToString(bestWorker)
	//d.outboundMsgs <- []string{
	//	strIdentity,
	//	PROTO_TASK,
	//	taskUUID.String(),
	//	msg.method,
	//  msg.params,
	//}
	taskMsg := []string{
		strIdentity,
		PROTO_TASK,
		taskUUID.String(),
		msg.method,
		msg.params,
	}
	go d.sendToOutbound(taskMsg)

	// setup cahanel listener for response with timeout
	// TODO: Check chanels for existance etc.
	select {
	case response := <-chResult:
		//fmt.Printf("ExecuteMethod got result %s", response)
		chResponse <- response
		//fmt.Println("ExecuteMethod write response")
	case <-time.After(time.Second * 2):
		d.removeTasks([]TaskId{taskId})
		fmt.Println("ExecuteMethod timeout")
	}

}

func (d *Dispatcher) RemoteCall(methodName string, params string, timeout time.Duration) (string, error) {

	var err error = nil
	// var result string

	chResult := make(chan string)

	apiMsg := ApiMessage{
		method: methodName,
		params: params,
	}

	go GrossDispatcher.ExecuteMethod(&apiMsg, chResult)

	select {
	case result, ok := <-chResult:
		if !ok {
			err = errors.New("Chanel closed")
		}
		return result, err

	case <-time.After(timeout):
		err = errors.New("RemoteCall timeout reached.")
		return "", err
	}

}
