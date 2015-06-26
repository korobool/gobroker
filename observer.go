package main

import (
	"fmt"
	"strings"
	"time"
)

// func ObserveMicroservices() {
// 	for {
// 		time.Sleep(time.Second)
// 	}
// }

// func StartMicroservicesObserving() {
// 	go ObserveMicroservices()
// }

func remoteCall(apiMsg *ApiMessage, chResult chan string) {

	go dispatcher.ExecuteMethod(apiMsg)

	select {
	case chResult <- strings.ToUpper(apiMsg.params):

	case <-time.After(time.Second * 2):
		fmt.Println("timeout")
	}
}
