package main

import (
	"bufio"
	"fmt"
	tcrsa "github.com/niclabs/tcrsa"
	zmq "github.com/pebbe/zmq4"
	"log"
	mathrand "math/rand"
	"os"
	"strconv"
	"time"
)

const ACS_COIN = "ACS_COIN"
const ACS_RBC = "ACS_RBC"
const ACS_ABA = "ACS_ABA"
const TPKE = "TPKE"

var all_ports = []string{"5000", "5010", "5020", "5030"}

var socket *zmq.Socket
var serverPort string

type honeybadger struct {
	sid                string
	pid                int
	B                  int
	N                  int
	f                  int
	sPK                tcrsa.KeyMeta
	sSK                tcrsa.KeyShare
	ePK                []byte
	eSK                []byte
	send               chan []byte
	recv               chan []byte
	round              int
	transaction_buffer []string
}

type broadcastIface func([]string, string, interface{})

type sendMessagesIface func(string, interface{})

type getCoinIface func(r int)

func (hb *honeybadger) submit_tx(tx string) {
	hb.transaction_buffer = append(hb.transaction_buffer, tx)
}

func Shuffle(vals []string) []string {
	r := mathrand.New(mathrand.NewSource(time.Now().Unix()))
	ret := make([]string, len(vals))
	n := len(vals)
	for i := 0; i < n; i++ {
		randIndex := r.Intn(len(vals))
		ret[i] = vals[randIndex]
		vals = append(vals[:randIndex], vals[randIndex+1:]...)
	}
	return ret
}

func random_selection(transaction_buffer []string, B int, N int) []string {
	var shuffled = Shuffle(transaction_buffer[:B])
	ret := make([]string, B/N)
	var i int = 0
	for ; i < B/N; i++ {
		ret = append(ret, shuffled[i])
	}
	return ret
}

func remove(txns []string, txn string) []string {
	var idx = -1
	for i := 0; i < len(txns); i++ {
		if txns[i] == txn {
			idx = i
		}
	}
	if idx != -1 {
		return append(txns[:idx], txns[idx+1:]...)
	}
	return txns
}

func recvMessages(zctx *zmq.Context, port string) {
	// Server port that listens for messages
	s, _ := zctx.NewSocket(zmq.REP)
	s.Bind("tcp://*:" + port)

	for {
		// Wait for next request from client
		msg, _ := s.Recv(0)
		log.Printf("Received %s\n", msg)

		// Do some 'work'
		time.Sleep(time.Second * 1)
	}
}

func sendMessages(port string, msg interface{}) {
	//Client port that sends messages
	socket.Connect("tcp://localhost:" + port)
	socket.Send(msg.(string), 0)
}

func broadcast(msg interface{}) {
	for i := 0; i < len(all_ports); i++ {
		if all_ports[i] != serverPort {
			sendMessages(all_ports[i], msg)
		}
	}
}

func (hb *honeybadger) run_round(r int, txn string, hb_block chan []string) {

	coin_recvs := make([](chan string), hb.N)
	aba_recvs := make([](chan string), hb.N)
	rbc_recvs := make([](chan string), hb.N)

	aba_inputs := make([](chan int), hb.N)
	aba_outputs := make([](chan int), hb.N)
	rbc_outputs := make([](chan string), hb.N)

	my_rbc_input := make(chan string)

	setup := func(j int) {
		coin_bcast := func(o int) {
			broadcast("ACS_COIN" + strconv.Itoa(j) + strconv.Itoa(o))
		}
		coin_recvs[j] = make(chan string)

		aba_bcast := func(o int) {
			broadcast("ACS_ABA" + strconv.Itoa(j) + strconv.Itoa(o))
		}
		aba_recvs[j] = make(chan string)
		go binaryagreement(hb.sid+"ABA"+strconv.Itoa(j), hb.pid, hb.N, hb.f, aba_inputs[j], aba_outputs[j], broadcast, aba_recvs[j],
			               hb.sid+"COIN"+strconv.Itoa(j), hb.pid, hb.N, hb.f, hb.sPK, hb.sSK, coin_bcast, coin_recvs[j])

		rbc_send := func(k int, o int) {
			sendMessages(all_ports[k], "ACS_RBC"+strconv.Itoa(j)+strconv.Itoa(o))
		}

		rbc_recvs[j] = make(chan string)
		go reliablebroadcast(hb.sid, hb.pid, hb.N, hb.f, j, my_rbc_input, rbc_recvs[j], rbc_send, rbc_outputs[j])
	}

	var j int
	for j = 0; j < hb.N; j++ {
		setup(j)
	}

	tpke_bcast := func(o int) {
		broadcast("ACS_RBC" + strconv.Itoa(0) + strconv.Itoa(o))
	}

	tpke_recv := make(chan string)

	rbc_values := make([]chan string, hb.N)
	go commonsubset(hb.pid, hb.N, hb.f, rbc_outputs, aba_inputs, aba_outputs, rbc_values)
	input := make(chan string)
	input <- txn

	honeybadgerBlock(hb.pid, hb.N, hb.f, input, my_rbc_input, rbc_values, hb_block)
}

func (hb *honeybadger) run() {
	var new_txns []string
	var hb_block chan []string
	for {
		var proposed = random_selection(hb.transaction_buffer, hb.B, hb.N)

		hb.run_round(hb.round, proposed[0], hb_block)

		new_txns = <-hb_block

		for i := 0; i < len(new_txns); i++ {
			hb.transaction_buffer = remove(hb.transaction_buffer, new_txns[i])
		}

		new_txns = nil

		hb.round += 1
	}
}

func main() {
	// Statically init all ports
	// all_ports := []string{"5000", "5010", "5020", "5030"}

	// Read ports from config file
	file, err := os.Open("ports.config")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		serverPort = scanner.Text()
		break
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	zctx, _ := zmq.NewContext()

	go recvMessages(zctx, serverPort)

	socket, _ = zctx.NewSocket(zmq.REQ)
	if serverPort != "5000" {
		sendMessages("5000", "Random message from "+serverPort)
	}

	for {

	}

	fmt.Println("Exiting")

}
