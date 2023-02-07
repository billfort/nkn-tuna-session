package test

import (
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	ts "github.com/nknorg/nkn-tuna-session"
)

const seedHex = "e68e046d13dd911594576ba0f4a196e9666790dc492071ad9ea5972c0b940435"
const listenerId = "Bob"
const dialerId = "Alice"

var listenerAddr string

func StartTunaListner(numListener int, ch chan string) (tunaSess *ts.TunaSessionClient) {
	acc, wal, err := CreateAccountAndWallet(seedHex)
	mc, err := CreateMultiClient(acc, listenerId, 2)
	tunaSess, err = CreateTunaSession(acc, wal, mc)

	listenerAddr = listenerId + "." + strings.SplitN(tunaSess.Addr().String(), ".", 2)[1]
	fmt.Println("listenerAddr ", listenerAddr)

	err = tunaSess.Listen(nil)
	if err != nil {
		log.Fatal(err)
	}

	go func(ch chan string) {
		for {
			ncpSess, err := tunaSess.Accept()
			if err != nil {
				log.Fatal(err)
			}
			log.Println(tunaSess.Addr(), "accepted a session")

			sessions := tunaSess.GetSessions()
			for key := range sessions {
				ch <- key
			}

			go func(c net.Conn) {
				err := read(c)
				if err != nil {
					log.Fatal(err)
				}
				c.Close()
			}(ncpSess)
		}
	}(ch)

	return
}

func StartTunaDialer(numBytes int, ch chan string) (tunaSess *ts.TunaSessionClient) {
	acc, wal, _ := CreateAccountAndWallet(seedHex)
	mc, _ := CreateMultiClient(acc, dialerId, 2)

	tunaSess, _ = CreateTunaSession(acc, wal, mc)

	diaConfig := CreateDialConfig(5000)
	ncpSess, err := tunaSess.DialWithConfig(listenerAddr, diaConfig)
	if err != nil {
		log.Fatal(err)
	}
	log.Println(tunaSess.Addr(), "dialed a session")
	sessions := tunaSess.GetSessions()
	for key := range sessions {
		ch <- key
	}

	go func() {
		err := write(ncpSess, numBytes)
		if err != nil {
			log.Fatal(err)
		}
		for {
			if ncpSess.IsClosed() {
				os.Exit(0)
			}
			time.Sleep(time.Millisecond * 100)
		}
	}()

	return
}

func CloseOneConn(tunaSess *ts.TunaSessionClient, sessKey string, connId string) {
	fmt.Printf("Going to close %v session %v conn %v\n", tunaSess.Addr(), sessKey, connId)
	conns := tunaSess.GetConns(sessKey)
	conn, ok := conns[connId]
	if ok {
		conn.Conn.Close()
		fmt.Printf("Closed %v session %v conn %v\n", tunaSess.Addr(), sessKey, connId)
	}
}
