package test

import (
	"testing"
	"time"

	ts "github.com/nknorg/nkn-tuna-session"
)

// go test -v -run=TestListener
func TestListener(t *testing.T) {
	var listenSess *ts.TunaSessionClient
	ch := make(chan string, 1)

	go func() {
		listenSess = StartTunaListner(2, ch)
	}()

	sessKey := <-ch
	time.Sleep(time.Second)
	CloseOneConn(listenSess, sessKey, "1")

	<-ch
}

// go test -v -run=TestDialer
func TestDialer(t *testing.T) {
	bytesToSend := 8 << 20
	var dialSess *ts.TunaSessionClient
	ch := make(chan string, 1)

	go func() {
		// wait for Listener be ready
		time.Sleep(2 * time.Second)
		dialSess = StartTunaDialer(bytesToSend, ch)
	}()

	sessKey := <-ch
	time.Sleep(time.Second)
	CloseOneConn(dialSess, sessKey, "1")

	<-ch
}
