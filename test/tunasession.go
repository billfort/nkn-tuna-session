package test

import (
	"fmt"
	"log"
	"strings"

	ts "github.com/nknorg/nkn-tuna-session"
)

const seedHex = "e68e046d13dd911594576ba0f4a196e9666790dc492071ad9ea5972c0b940435"
const listenerId = "Bob"
const dialerId = "Alice"

const remoteAddr = "Bob.be285ff9330122cea44487a9618f96603fde6d37d5909ae1c271616772c349fe"

func StartTunaListner(numListener int, ch chan string) (tunaSess *ts.TunaSessionClient) {
	acc, wal, err := CreateAccountAndWallet(seedHex)
	mc, err := CreateMultiClient(acc, listenerId, 2)
	tunaSess, err = CreateTunaSession(acc, wal, mc)
	tunaSess.SetName(listenerId)

	listenerAddr := listenerId + "." + strings.SplitN(tunaSess.Addr().String(), ".", 2)[1]
	fmt.Println("listenerAddr ", listenerAddr)

	err = tunaSess.Listen(nil)
	if err != nil {
		log.Fatal("tunaSess.Listen ", err)
	}

	ncpSess, err := tunaSess.Accept()
	if err != nil {
		log.Fatal("tunaSess.Accept ", err)
	}
	log.Println(tunaSess.Name, "accepted a session")
	sesss := tunaSess.GetSessions()
	for key, sess := range sesss {
		if sess == ncpSess {
			ch <- key
		}
	}

	go func() {
		err = read(ncpSess)
		if err != nil {
			fmt.Printf("StartTunaListner read err:%v\n", err)
		} else {
			fmt.Printf("Finished reading, close ncp.session now\n")
		}
		ncpSess.Close()
		ch <- "end"
	}()

	return
}

func StartTunaDialer(numBytes int, ch chan string) (tunaSess *ts.TunaSessionClient) {
	acc, wal, _ := CreateAccountAndWallet(seedHex)
	mc, _ := CreateMultiClient(acc, dialerId, 2)

	tunaSess, _ = CreateTunaSession(acc, wal, mc)
	tunaSess.SetName(dialerId)

	diaConfig := CreateDialConfig(5000)
	ncpSess, err := tunaSess.DialWithConfig(remoteAddr, diaConfig)
	if err != nil {
		log.Fatal("tunaSess.DialWithConfig ", err)
	}
	log.Println(tunaSess.Name, "dialed a session")
	sesss := tunaSess.GetSessions()
	for key, sess := range sesss {
		if sess == ncpSess {
			ch <- key
		}
	}

	go func() {
		err = write(ncpSess, numBytes)
		if err != nil {
			fmt.Printf("StartTunaDialer write err:%v\n", err)
		} else {
			fmt.Printf("Finished reading, close ncp.session now\n")
		}
		ncpSess.Close()
		ch <- "end"
	}()

	return
}

func CloseOneConn(tunaSess *ts.TunaSessionClient, sessKey string, connId string) {
	conns := tunaSess.GetConns(sessKey)
	conn, ok := conns[connId]
	if ok {
		conn.Conn.Close()
		fmt.Printf("%v conn %v is closed by program\n", tunaSess.Name, connId)
	}
}
