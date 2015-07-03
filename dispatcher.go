package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/korobool/gomixer/proto"
	"github.com/m4rw3r/uuid"
	zmq "github.com/pebbe/zmq4"
	"sync"
	"time"
)

const (
	OutBufferSize        = 10
	MaxWorkers           = 16
	AliveTimeout         = 2
	MaxKAFailed          = 2
	HeartbeatingInterval = 1 * time.Second
	WorkerMaxTasks       = 200
	ExecuteTimeout       = 2 * time.Second
	PollInterval         = 500 * time.Millisecond
)

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
	zmqPullSocket *zmq.Socket
	zmqPullPoller *zmq.Poller
	zmqPushSocket *zmq.Socket
	zmqPushPoller *zmq.Poller
	locks         Locks
	//TODO: synchronize this map
	//use this datastructure https://github.com/streamrail/concurrent-map
	tasks       map[TaskId]*Task
	pullOutMsgs chan []string
	pushOutMsgs chan []string
	workers     map[uint32]*WorkerInfo
	methods     map[string][]uint32
	ids         map[string][MaxWorkers]bool
	accepts     map[uint32]chan struct{}
}

func createZmqSocket(uri string) (*zmq.Socket, *zmq.Poller, error) {

	zmqSocket, err := zmq.NewSocket(zmq.ROUTER)
	if err != nil {
		return nil, nil, err
	}
	if err := zmqPullSocket.Bind(uri); err != nil {
		return nil, nil, err
	}
	zmqPoller := zmq.NewPoller()
	zmqPoller.Add(zmqSocket, zmq.POLLIN)

	return zmqSocket, zmqPoller, nil

}

func NewDispatcher(uriPull string, uriPush string) (*Dispatcher, error) {

	zmqPullSocket, zmqPullPoller, err := createZmqSocket(uriPull)
	if err != nil {
		return nil, err
	}
	zmqPushSocket, zmqPushPoller, err := createZmqSocket(uriPush)
	if err != nil {
		return nil, err
	}

	return &Dispatcher{
		zmqPullSocket: zmqPullSocket,
		zmqPullPoller: zmqPullPoller,
		zmqPushSocket: zmqPushSocket,
		zmqPushPoller: zmqPushPoller,
		locks:         Locks{workers: new(sync.RWMutex), tasks: new(sync.Mutex)},
		tasks:         make(map[TaskId]*Task),
		pullOutMsgs:   make(chan []string),
		pushOutMsgs:   make(chan []string),
		//pullOutMsgs:   make(chan []string, OutBufferSize),
		//pushOutMsgs:   make(chan []string, OutBufferSize),
		workers: make(map[uint32]*WorkerInfo),
		methods: make(map[string][]uint32),
		ids:     map[string][MaxWorkers]bool{},
		accepts: make(map[uint32]chan struct{}),
	}, err
}

func (d *Dispatcher) run() {
	go d.ZmqPushLoopRun()
	go d.ZmqPullLoopRun()
	go d.HeartbeatingRun()
}

func (d *Dispatcher) HeartbeatingRun() {
	for {

		removeList := []uint32{}

		fmt.Println("==================")
		for i, w := range d.workers {
			fmt.Printf(">>>worker: [%d] <%s:%d>\t%d\n", i, w.workerType, w.workerId, len(w.tasks))
		}
		fmt.Println("==================")

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

			kaMsg := []string{strIdentity, proto.KA, fmt.Sprintf("%d", worker.workerId)}

			go d.sendToPush(kaMsg, DefaultTimeout)

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

func (d *Dispatcher) removeWorkerTask(identity uint32, taskUUID uuid.UUID) {
	taskId := TaskId{identity, taskUUID}
	worker := d.workers[identity]

	for i, task := range worker.tasks {
		if task == taskId {
			freshTaskList := []TaskId{}
			freshTaskList = append(worker.tasks[:i], worker.tasks[i+1:]...)
			d.workers[identity].tasks = freshTaskList
		}
	}
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

func (d *Dispatcher) sendToPush(msg []string, timeout time.Duration) {
	select {
	case d.pullOutMsgs <- msg:
	case <-time.after(timeout):
	}
}

func (d *Dispatcher) sendToPull(msg []string, timeout time.Duration) {
	select {
	case d.pullOutMsgs <- msg:
	case <-time.after(timeout):
	}
}

func (d *Dispatcher) registerInMethods(identity uint32, exposedMethods []string) {

	for idx := range exposedMethods {
		method := exposedMethods[idx]
		if _, ok := d.methods[method]; ok {
			d.methods[method] = append(d.methods[method], identity)
		} else {
			d.methods[method] = []uint32{identity}
		}
	}
}

func (d *Dispatcher) attachWorker(identity, msg []string, timeout time.Duration) {

	workerType := msg[2]

	workerId, err := d.takeWorkerId(workerType)
	if err != nil {
		fmt.Println("failed to attach new worker: no free workerId")
		return
	}

	strIdentity := identityIntToString(identity)
	acceptMsg := []string{strIdentity, proto.ACCEPT, fmt.Sprintf("%d", workerId)}

	chAccept := make(chan struct{})

	d.accepts[identity] = chAccept

	go sendToPull(acceptMsg, timeout)

	select {
	case <-chAccept:
	case <-time.after(timeout):
		d.releaseWorkerId(workerId)
		return
	}

	worker = &WorkerInfo{
		workerId:   workerId,
		workerType: workerType,
		tasks:      []TaskId{},
		kaLast:     0,
		kaFailed:   0,
	}

	//WARN: remove locks in removeWorker()!!!!
	d.locks.workers.Lock()
	if _, ok := d.workers[identity]; ok {
		d.removeWorker(identity)
	}
	d.workers[identity] = worker
	// Register methods exposed by a worker
	d.registerInMethods(msg[3:])
	d.locks.workers.Unlock()
}

func (d *Dispatcher) recvPull(msg []string) error {

	// 4 bytes to int32 conversion (msg[0][1:] has length 4)
	identity := identityByteStrToInt(msg[0][1:])

	//fmt.Printf("[id:%d] recieved: %q\n", identity, msg)

	cmd := msg[1]
	if cmd == proto.READYPUSH {
		go d.attachWorker(identity, DefaultTimeout)

	}
	// Ignore all the messages with unknown identity
	if _, ok := d.workers[identity]; !ok {
		fmt.Println("Unknown identity: ", identity)
		return nil
	}

	if msg[1] == proto.DONE {

		if len(msg) < 3 {
			return errors.New("Malformed message")
		}

		taskUUID, err := uuid.FromString(msg[2])
		if err != nil {
			return errors.New("Wrong UUID in message")
		}

		if task, ok := d.tasks[TaskId{identity, taskUUID}]; ok {
			// TODO: check if channel closed
			go func() { task.chResult <- msg[3] }()
		}
	}
	return nil
}

func (d *Dispatcher) recvPush(msg []string) error {

	// 4 bytes to int32 conversion (msg[0][1:] has length 4)
	identity := identityByteStrToInt(msg[0][1:])

	//fmt.Printf("[id:%d] recieved: %q\n", identity, msg)

	cmd := msg[1]
	if cmd == proto.READYPULL {
		if chAccept, ok := d.accepts[identity]; ok {
			go func() { chAccept <- struct{}{} }()
		}
	}
	return nil
}

func sendFromChannel(chMsg chan []string, socket *zmq.Socket, stop chan struct{}) {
	for {
		select {
		case msg := <-chMsg:
			socket.SendMessage(msg)
		case <-stop:
			return
		}
	}
}

func (d *Dispatcher) ZmqPushLoopRun() error {

	chStop := make(chan struct{})

	for {

		go sendFromChannel(d.pushOutMsgs, d.zmqPushSocket, chStop)

		<-time.Tick(PollInterval)
		chStop <- struct{}{}

		// Replce hardcoded value with some reasonable iteration number
		for i := 0; i < 20; i++ {
			sockets, err := d.zmqPushPoller.Poll(0)
			if err != nil {
				fmt.Println(err)
				return err //  Interrupt loop
			}

			if len(sockets) != 1 {
				break
			}

			msg, err := d.zmqPushSocket.RecvMessage(zmq.DONTWAIT)
			if err != nil {
				fmt.Println(err)
				break
			} else {
				go d.recvPush(msg)
			}
		}
	}

	return nil
}

func (d *Dispatcher) ZmqPullLoopRun() error {

	chStop := make(chan struct{})

	for {

		for {
			select {
			case <-time.After(PollInterval):
				break
			default:
				sockets, err := d.zmqPushPoller.Poll(PollInterval)
				if err != nil {
					fmt.Println(err)
					return err //  Interrupt loop
				}

				if len(sockets) != 1 {
					break
				}
				msg, err := d.zmqPullSocket.RecvMessage(zmq.DONTWAIT)
				if err != nil {
					fmt.Println(err)
				} else {
					go d.recvPull(msg)
				}
			}
		}

		for i := 0; i < 20; i++ {
			select {
			case outMsg := <-d.pullOutMsgs:
				d.zmqPullSocket.SendMessage(msg)
			default:
			}
		}
	}

	return nil
}

func (d *Dispatcher) getBestWorker(methodName string) (uint32, error) {

	candidates := d.methods[methodName]
	shortest := int(^uint(0) >> 1)
	idx := 0

	// TODO: check for safity and put a Mutex if needed.
	for index, candidate := range candidates {
		//fmt.Printf("len: %d shortest: %d idx: %d\n",
		//				len(d.workers[candidate].tasks), shortest, idx)
		if len(d.workers[candidate].tasks) < shortest {
			shortest = len(d.workers[candidate].tasks)
			idx = index
		}
	}

	if len(candidates) == 0 || shortest > WorkerMaxTasks {
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
		fmt.Println(err)
		close(chResponse)
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

	taskMsg := []string{
		strIdentity,
		proto.TASK,
		taskUUID.String(),
		msg.method,
		msg.params,
	}
	go d.sendToPush(taskMsg, ExecuteTimeout)

	// setup cahanel listener for response with timeout
	// TODO: Check chanels for existance etc.
	select {
	case response := <-chResult:
		//fmt.Printf("ExecuteMethod got result %s", response)
		d.removeTasks([]TaskId{taskId})
		d.removeWorkerTask(bestWorker, taskUUID)
		chResponse <- response
		//fmt.Println("ExecuteMethod write response")
	case <-time.After(ExecuteTimeout):
		d.removeTasks([]TaskId{taskId})
		d.removeWorkerTask(bestWorker, taskUUID)
		fmt.Println("ExecuteMethod timeout")
	}

}

func (d *Dispatcher) RemoteCall(methodName string, params []byte, timeout time.Duration) ([]byte, error) {

	var err error = nil

	chResult := make(chan string)

	apiMsg := ApiMessage{
		method: methodName,
		params: string(params),
	}

	go GrossDispatcher.ExecuteMethod(&apiMsg, chResult)

	select {
	case result, ok := <-chResult:
		if !ok {
			err = errors.New("Chanel closed")
		}
		return []byte(result), err

	case <-time.After(timeout):
		err = errors.New("RemoteCall timeout reached.")
		return []byte{}, err
	}

}
