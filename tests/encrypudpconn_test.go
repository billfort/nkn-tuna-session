package tests

import (
	"errors"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nknorg/ncp-go"
	ts "github.com/nknorg/nkn-tuna-session"
	"github.com/nknorg/tuna"
)

// go test -v -run=TestEncryptUdpListener
func TestEncryptUdpListener(t *testing.T) {
	ch := make(chan string, 1)
	go func() {
		StartEncryptUDPListner(numUdpListener, ch)
	}()
	<-ch
}

// go test -v -run=TestMultiEncryptUdpDialer
func TestMultiEncryptUdpDialer(t *testing.T) {
	ch := make(chan string, 1)
	go func() {
		// wait for Listener be ready
		time.Sleep(2 * time.Second)
		StartMultiUDPDialer(numUdpListener, ch)
	}()
	<-ch
}

func StartEncryptUDPListner(numListener int, ch chan string) (tunaSess *ts.TunaSessionClient, ncpSess *ncp.Session) {
	acc, wal, err := CreateAccountAndWallet(seedHex)
	if err != nil {
		log.Fatal("CreateAccountAndWallet err: ", err)
	}
	mc, err := CreateMultiClient(acc, listenerId, 2)
	if err != nil {
		log.Fatal("CreateMultiClient err: ", err)
	}
	tunaSess, err = CreateTunaSession(acc, wal, mc, numListener)
	if err != nil {
		log.Fatal("CreateTunaSession err: ", err)
	}

	err = tunaSess.Listen(nil)
	if err != nil {
		log.Fatal("tunaSess.Listen ", err)
	}
	go func() { // TCP
		sess, err := tunaSess.Accept()
		if err != nil {
			log.Fatal("tunaSess.Accept tcp ", err)
		}
		ncpSess = sess.(*ncp.Session)

		go func() {
			err = readTcp(ncpSess)
			if err != nil {
				log.Printf("StartTunaUDPListner TCP read err:%v\n", err)
			} else {
				log.Printf("Finished TCP reading, close ncp.session now\n")
			}
			ncpSess.Close()
		}()
	}()

	udpConn, err := tunaSess.ListenUDP(nil)
	if err != nil {
		log.Println("ListenUDP err ", err)
	}
	go func() {
		err = readEncryptUdp(udpConn)
		if err != nil {
			log.Printf("StartTunaUDPListner UDP read err:%v\n", err)
		} else {
			log.Printf("StartTunaUDPListner finished UDP reading, close udpConn now\n")
		}
		udpConn.Close()
		close(ch)
	}()

	return
}

func StartMultiUDPDialer(numListener int, ch chan string) {
	acc, wal, err := CreateAccountAndWallet(seedHex)
	if err != nil {
		log.Fatal("CreateAccountAndWallet err: ", err)
	}
	mc, err := CreateMultiClient(acc, dialerId, 2)
	if err != nil {
		log.Fatal("CreateMultiClient err: ", err)
	}

	diaConfig := CreateDialConfig(5000)
	var wg sync.WaitGroup
	for i := 0; i < numUdpDialers; i++ {
		fmt.Printf("start dialer %v now\n", i)
		wg.Add(1)
		go func(dialerNum int) {
			defer wg.Done()

			tunaSess, err := CreateTunaSession(acc, wal, mc, numListener)
			if err != nil {
				log.Fatal("CreateTunaSession err: ", err)
			}

			udpConn, err := tunaSess.DialUDPWithConfig(remoteAddr, diaConfig)
			if err != nil {
				log.Fatal("StartMultiUDPDialer.DialUDPWithConfig err ", err)
			}
			fmt.Printf("dialer %v DialUDPWithConfig successfully, going to write\n", dialerNum)

			err = writeEncryptUdp(udpConn, dialerNum)
			if err != nil {
				log.Printf("StartMultiUDPDialer Dialer %v  write err:%v\n", dialerNum, err)
			} else {
				log.Printf("UDP Dialer %v Finished UDP writing", dialerNum)
			}
			time.Sleep(2 * time.Second) // wait for reader to read data.
			log.Printf("UDP Dialer %v close udpConn now", dialerNum)
			udpConn.Close()
		}(i)

		time.Sleep(time.Second)
	}

	wg.Wait()
	close(ch)
}

func readEncryptUdp(udpConn *tuna.EncryptUDPConn) error {
	timeStart := time.Now()
	bytesReceived := make(map[int]int) // dialer number to received bytes.
	nFull := 0
	var mu sync.RWMutex
	for {
		b := make([]byte, bufSize)
		n, _, err := udpConn.ReadFromUDP(b)
		if err != nil {
			return err
		}

		msg := string(b[:n])
		arr := strings.Split(msg, ":")
		dialerNum, _ := strconv.Atoi(arr[0])

		mu.RLock()
		recved := bytesReceived[dialerNum]
		mu.RUnlock()

		recved += n
		mu.Lock()
		bytesReceived[dialerNum] = recved
		mu.Unlock()

		if strings.Compare(msg, msgs[dialerNum]) != 0 {
			fmt.Printf("\ndialer %v, received len: %v, data: %v, it is expected len: %v data: %v.\n\n",
				dialerNum, len(msg), msg, len(msgs[dialerNum]), msgs[dialerNum])
			return errors.New("wrong message")
		}

		if ((recved - n) * 10 / bytesToSend) != (recved * 10 / bytesToSend) {
			log.Printf("udp conn %v received %v bytes %.3v MB/s", dialerNum, recved,
				float64(recved)/math.Pow(2, 20)/(float64(time.Since(timeStart))/float64(time.Second)))
		}
		if recved >= bytesToSend {
			log.Printf("udp conn %v finish receiving %v bytes", dialerNum, recved)
			nFull++
		}
		if nFull == numUdpDialers {
			break
		}
	}
	return nil
}

var msgs = []string{"0:00000000", "1:111111111111111111", "2:2222222222222222222222222222",
	"3:33333333333333333333333333333333333333"}

func writeEncryptUdp(udpConn *tuna.EncryptUDPConn, dialerNum int) error {
	timeStart := time.Now()

	bytesSent := 0
	for {
		n, _, err := udpConn.WriteMsgUDP([]byte(msgs[dialerNum]), nil, nil)
		if err != nil {
			log.Printf("dialer %v udpConn.WriteMsgUDP err %v\n", dialerNum, err)
			time.Sleep(2 * time.Second)
			continue
		}
		if n != len(msgs[dialerNum]) {
			fmt.Printf("\ndialer %v write len %v is not eaqul to %v\n\n", dialerNum, n, len(msgs[dialerNum]))
			continue
		}
		bytesSent += n
		if bytesSent >= bytesToSend {
			log.Printf("dialer %v finish sending %v bytes %.3v MB/s",
				dialerNum, bytesSent, float64(bytesSent)/math.Pow(2, 20)/(float64(time.Since(timeStart))/float64(time.Second)))
			break
		} else {
			if ((bytesSent - n) * 10 / bytesToSend) != (bytesSent * 10 / bytesToSend) {
				log.Printf("dialer %v sent %v bytes %.3v MB/s", dialerNum, bytesSent,
					float64(bytesSent)/math.Pow(2, 20)/(float64(time.Since(timeStart))/float64(time.Second)))
			}
		}
		time.Sleep(udpWriteInterval)
	}
	return nil
}
