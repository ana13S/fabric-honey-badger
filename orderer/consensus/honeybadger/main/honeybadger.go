package main

import (
	tcrsa "github.com/niclabs/tcrsa"
	zmq "github.com/pebbe/zmq4"
	"log"
	mathrand "math/rand"
	"os"
	"strconv"
	"strings"
	"threshsig"
	"time"
)

const ACS_COIN = "ACS_COIN"
const ACS_RBC = "ACS_RBC"
const ACS_ABA = "ACS_ABA"
const TPKE = "TPKE"

var all_ports = []string{"5000", "5010", "5020", "5030"}

var zctx *zmq.Context
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
	send               chan []byte
	recv               chan []byte
	round              int
	transaction_buffer []string
}

type hbMessage struct {
	msgType string
	sender  int
	msg     string
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

func sendMessages(port string, hbm hbMessage) {
	//Client port that sends messages
	socket.Connect("tcp://localhost:" + port)
	var finalMessage string = hbm.msgType + "_" + strconv.Itoa(hbm.sender) + "_" + hbm.msg
	socket.Send(finalMessage, 0)
}

func send(pid int, msg hbMessage) {
	sendMessages(all_ports[pid], msg)
}

func broadcast(hbm hbMessage) {
	for i := 0; i < len(all_ports); i++ {
		if all_ports[i] != serverPort {
			sendMessages(all_ports[i], hbm)
		}
	}
}

func getChannelFromMsg(
	msgType string,
	sender int,
	coin_recvs []chan string,
	aba_recvs []chan string,
	rbc_recvs []chan string,
) chan string {
	if msgType == "ABA" {
		return aba_recvs[sender]
	} else if msgType == "COIN" {
		return coin_recvs[sender]
	} else if msgType == "RBC" {
		return rbc_recvs[sender]
	} else {
		panic("Unknown message type")
	}
}

func broadcast_receiver(coin_recvs []chan string, aba_recvs []chan string, rbc_recvs []chan string) {
	s, _ := zctx.NewSocket(zmq.REP)
	s.Bind("tcp://*:" + serverPort)

	for {
		// Wait for next request from client
		message, _ := s.Recv(0)
		log.Printf("Received %s\n", message)

		splitMsg := strings.Split(message, "_")
		msgType := splitMsg[0]
		sender, _ := strconv.Atoi(splitMsg[1])
		msg := splitMsg[2]

		channel := getChannelFromMsg(msgType, sender, coin_recvs, aba_recvs, rbc_recvs)

		channel <- msg

		// Do some 'work'
		time.Sleep(time.Second * 1)
	}
}

func (hb *honeybadger) run_round(r int, txn string, hb_block chan []string) {

	sid := hb.sid + ":" + strconv.Itoa(r)

	coin_recvs := make([](chan string), hb.N)
	aba_recvs := make([](chan string), hb.N)
	rbc_recvs := make([](chan string), hb.N)

	aba_inputs := make([](chan int), hb.N)
	aba_outputs := make([](chan int), hb.N)
	rbc_outputs := make([](chan string), hb.N)

	my_rbc_input := make(chan string)

	setup := func(j int) {
		coin_recvs[j] = make(chan string)
		aba_recvs[j] = make(chan string)

		go binaryagreement(sid+"ABA"+strconv.Itoa(j), hb.pid, hb.N, hb.f, aba_inputs[j], aba_outputs[j], aba_recvs[j],
			sid+"COIN"+strconv.Itoa(j), hb.pid, hb.N, hb.f, hb.sPK, hb.sSK, coin_recvs[j])

		rbc_recvs[j] = make(chan string)

		go reliablebroadcast(sid+"RBC"+strconv.Itoa(j), hb.pid, hb.N, hb.f, j, my_rbc_input, rbc_recvs[j], rbc_outputs[j])
	}

	for j := 0; j < hb.N; j++ {
		setup(j)
	}

	rbc_values := make([]chan string, hb.N)

	go commonsubset(hb.pid, hb.N, hb.f, rbc_outputs, aba_inputs, aba_outputs, rbc_values)

	go broadcast_receiver(coin_recvs, aba_recvs, rbc_recvs)

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

		if hb.round == 10 {
			break
		}
	}
}

func main() {
	pid, _ := strconv.Atoi(os.Args[1])

	serverPort = all_ports[pid]

	zctx, _ = zmq.NewContext()

	socket, _ = zctx.NewSocket(zmq.REQ)

	var transaction_buffer = []string{"A_B_100", "B_C_200", "D_E_500", "E_B_300"}

	shares, meta := threshsig.Dealer(4, 2, 2048)

	hb := honeybadger{
		sid:                "sidA",
		pid:                pid,
		B:                  4,
		N:                  4,
		f:                  1,
		sPK:                *meta,
		sSK:                *shares[pid],
		round:              0,
		transaction_buffer: transaction_buffer,
	}

	hb.run()
}
