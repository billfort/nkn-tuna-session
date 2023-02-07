package test

import (
	"testing"
	"time"

	ts "github.com/nknorg/nkn-tuna-session"
)

func TestMain(m *testing.M) {
	bytesToSend := 8 << 20
	var listenSess, dialSess *ts.TunaSessionClient
	ch1 := make(chan string)
	ch2 := make(chan string)

	go func() {
		listenSess = StartTunaListner(2, ch1)
	}()

	go func() {
		// wait for Listener be ready
		time.Sleep(8 * time.Second)
		dialSess = StartTunaDialer(bytesToSend, ch2)
	}()

	sessKey1 := <-ch1
	time.Sleep(8 * time.Second)
	CloseOneConn(listenSess, sessKey1, "1")

	sessKey2 := <-ch2
	time.Sleep(time.Second)
	CloseOneConn(dialSess, sessKey2, "1")

	ch := make(chan int)
	<-ch
}
