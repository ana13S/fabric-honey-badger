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
	"strings"
	"sync"
	"threshsig"
	"time"
)

var all_ports = []string{"5000", "5010", "5020", "5030"}

var pid int
var zctx *zmq.Context
var socket *zmq.Socket
var serverPort string
var clients = make(map[int]*zmq.Socket)
var wg sync.WaitGroup

var coin_recvs [](chan string)
var aba_recvs [](chan string)
var rbc_recvs [](chan string)

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
	for i := 0; i < B/N; i++ {
		ret[i] = shuffled[i]
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

func getChannelFromMsg(
	msgType string,
	sender int,
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

func handleMessageToSelf(hbm hbMessage) {
	channel := getChannelFromMsg(hbm.msgType, pid)
	channel <- hbm.msg
}

func sendMessages(to int, hbm hbMessage) {
	//Client port that sends messages
	// socket.Connect("tcp://localhost:" + port)
	if pid != to {
		var finalMessage string = hbm.msgType + "_" + strconv.Itoa(hbm.sender) + "_" + hbm.msg
		fmt.Println("[sendMessages] Sending message ", finalMessage, " to ", to)
		clients[to].Send(finalMessage, 0)

		reply, _ := clients[to].Recv(0)
		fmt.Println("[sendMessages] Received ", reply)
	} else {
		handleMessageToSelf(hbm)
	}
}

func send(to int, msg hbMessage) {
	sendMessages(to, msg)
}

func broadcast(hbm hbMessage) {
	for i := 0; i < len(all_ports); i++ {
		sendMessages(i, hbm)
	}
}

func broadcast_receiver(s *zmq.Socket) {
	// s, _ := zctx.NewSocket(zmq.REP)
	// s.Bind("tcp://*:" + serverPort)

	for {
		// Wait for next request from client
		message, _ := s.Recv(0)
		fmt.Println("[broadcast_receiver] Received ", message)

		// Parse the message
		splitMsg := strings.Split(message, "_")
		msgType := splitMsg[0]
		sender, _ := strconv.Atoi(splitMsg[1])
		msg := splitMsg[2]

		channel := getChannelFromMsg(msgType, sender)

		// Put message in apt channel
		channel <- msg

		// Do some 'work'
		time.Sleep(time.Second * 1)

		// Send reply back to client
		fmt.Println("[broadcast_receiver] Sending empty string as reply")
		s.Send("", 0)
	}
}

func (hb *honeybadger) run_round(r int, txn string, hb_block chan []string, receiver *zmq.Socket) {
	fmt.Println("[run_round] Running round ", r, " proposed txn: ", txn)

	sid := hb.sid + ":" + strconv.Itoa(r)

	coin_recvs = make([](chan string), hb.N)
	aba_recvs = make([](chan string), hb.N)
	rbc_recvs = make([](chan string), hb.N)

	go broadcast_receiver(receiver)

	aba_inputs := make([](chan int), hb.N)
	aba_outputs := make([](chan int), hb.N)
	rbc_outputs := make([](chan string), hb.N)

	my_rbc_input := make(chan string, 1)

	setup := func(j int) {
		fmt.Println("[run_round] Setting up for node ", j)

		// These are supposed to be infinite sized channels. Initializing with
		// size N instead.
		coin_recvs[j] = make(chan string, hb.N)
		aba_recvs[j] = make(chan string, hb.N)

		aba_inputs[j] = make(chan int, 1)
		aba_outputs[j] = make(chan int, 1)
		rbc_outputs[j] = make(chan string, 1)

		fmt.Println("[run_round] Spawning binary agreement for node ", j)
		go binaryagreement(sid+"ABA"+strconv.Itoa(j), hb.pid, hb.N, hb.f, aba_inputs[j], aba_outputs[j], aba_recvs[j],
			sid+"COIN"+strconv.Itoa(j), hb.pid, hb.N, hb.f, hb.sPK, hb.sSK, coin_recvs[j])

		// These are supposed to be infinite sized channels. Initializing with
		// size N instead.
		rbc_recvs[j] = make(chan string, hb.N)

		fmt.Println("[run_round] Spawning reliable broadcast for node ", j)
		go reliablebroadcast(sid+"RBC"+strconv.Itoa(j), hb.pid, hb.N, hb.f, j, my_rbc_input, rbc_recvs[j], rbc_outputs[j])
	}

	for j := 0; j < hb.N; j++ {
		setup(j)
	}

	rbc_values := make([]chan string, hb.N)

	fmt.Println("[run_round] Spawning common subset")
	go commonsubset(hb.pid, hb.N, hb.f, rbc_outputs, aba_inputs, aba_outputs, rbc_values)

	fmt.Println("[run_round] Spawning broadcast receiver")
	go broadcast_receiver(receiver)

	fmt.Println("[run_round] Adding txn ", txn, " to input channel.")
	input := make(chan string, 1)
	input <- txn

	fmt.Println("[run_round] Calling honeybadger_block for round ", r)
	honeybadgerBlock(hb.pid, hb.N, hb.f, input, my_rbc_input, rbc_values, hb_block)
}

func (hb *honeybadger) run(receiver *zmq.Socket) {
	fmt.Println("Starting honeybadger")
	var new_txns []string
	var hb_block chan []string
	var proposed []string
	for round := 0; round < 1; round++ {
		proposed = random_selection(hb.transaction_buffer, hb.B, hb.N)
		fmt.Println("Proposal for round ", round, ": ", proposed)

		hb.run_round(round, proposed[0], hb_block, receiver)

		fmt.Println("[run] Round ", round, " is complete.")
		new_txns = <-hb_block
		fmt.Println("[run] Transactions committed in round ", round, ": ", new_txns)

		for i := 0; i < len(new_txns); i++ {
			hb.transaction_buffer = remove(hb.transaction_buffer, new_txns[i])
		}

		new_txns = nil
	}
}

func sync_nodes(s *zmq.Socket, pid int) {
	defer wg.Done()
	var receiver_map = make(map[string]string)
	var counter = 0

	for {
		// Wait for next request from client
		msg, _ := s.Recv(0)
		fmt.Println("[sync_nodes] Received ", msg)

		// Do some 'work'
		time.Sleep(time.Second * 1)

		// Send reply back to client
		fmt.Println("[sync_nodes] Sending World as reply")
		s.Send("World from "+strconv.Itoa(pid), 0)

		sender, ok := receiver_map[msg[len(msg)-1:]]
		if !ok && strings.Contains(msg, "Hello") {
			receiver_map[sender] = sender
			counter += 1

			if counter == 3 {
				break
			}
		}
	}
}

func main() {
	wg.Add(1)
	pid, _ = strconv.Atoi(os.Args[1])

	N := 4
	f := 1
	B := 4

	serverPort = all_ports[pid]

	file, err := os.Open("transactions.log")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	var transaction_buffer []string
	for scanner.Scan() {
		transaction_buffer = append(transaction_buffer, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	zctx, _ = zmq.NewContext()

	receiver, _ := zctx.NewSocket(zmq.REP)
	defer receiver.Close()

	receiver.Bind("tcp://*:" + all_ports[pid])

	go sync_nodes(receiver, pid)

	time.Sleep(5 * time.Second)

	fmt.Println("[main] buffer: ", transaction_buffer, " N: ", N, " f: ", f, " B: ", B)

	shares, meta := threshsig.Dealer(4, 3, 2048)

	hb := honeybadger{
		sid:                "sidA",
		pid:                pid,
		B:                  B,
		N:                  N,
		f:                  f,
		sPK:                *meta,
		sSK:                *shares[pid],
		transaction_buffer: transaction_buffer,
	}

	for i := 0; i < N; i++ {
		if i != pid {
			clients[i], _ = zctx.NewSocket(zmq.REQ)
			defer clients[i].Close()
			clients[i].Connect("tcp://localhost:" + all_ports[i])

			fmt.Println("[main] Sending hello to ", i)
			clients[i].Send("Hello from "+strconv.Itoa(pid), 0)

			msg, _ := clients[i].Recv(0)
			fmt.Printf("[main] Received reply from %d: %s\n", i, msg)
		}
	}

	time.Sleep(5 * time.Second)

	fmt.Println("[main] Waiting for sync go routines to complete")
	wg.Wait()

	time.Sleep(10 * time.Second)

	fmt.Println("[main] Ready to run honeybadger ", hb.N)
	hb.run(receiver)
}
