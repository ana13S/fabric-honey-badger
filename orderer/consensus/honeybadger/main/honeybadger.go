package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/hpcloud/tail"
	"github.com/juju/fslock"
	tcrsa "github.com/niclabs/tcrsa"
	zmq "github.com/pebbe/zmq4"
	"log"
	mathrand "math/rand"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"threshsig"
	"time"
)

var all_ports = []string{"5000", "5010", "5020", "5030"}
var files = []string{"5000.txt", "5010.txt", "5020.txt", "5030.txt"}
var fileLocks = make([]*fslock.Lock, 4)
var writeFileHandlers = make([](*os.File), 4)
var readFileHandler *os.File
var readScanner *bufio.Scanner

var pid int
var zctx *zmq.Context
var socket *zmq.Socket
var serverPort string
var clients = make(map[int]*zmq.Socket)
var wg sync.WaitGroup
var currentRound int

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
	round   int
	msgType string
	channel int
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
	r := mathrand.New(mathrand.NewSource(time.Now().Unix()))
	randIndex := r.Intn(B)
	return []string{transaction_buffer[randIndex]}
}

func remove(txns []string, txn string) []string {
	var idx = -1
	pattern := regexp.MustCompile(`\s+`)
	for i := 0; i < len(txns); i++ {
		if strings.ToLower(pattern.ReplaceAllString(txns[i], " ")) == strings.ToLower(pattern.ReplaceAllString(txn, " ")) {
			idx = i
			break
		}
	}
	if idx != -1 {
		return append(txns[:idx], txns[idx+1:]...)
	} else {
		fmt.Println("[remove] Did not find txn: ", txn, " in txns: ", txns)
	}
	return txns
}

func getChannelFromMsg(
	msgType string,
	channel int,
) chan string {
	if msgType == "ABA" {
		return aba_recvs[channel]
	} else if msgType == "COIN" {
		return coin_recvs[channel]
	} else if msgType == "RBC" {
		return rbc_recvs[channel]
	} else {
		panic("Unknown message type")
	}
}

func sendMessages(to int, hbm hbMessage) {
	var finalMessage string = strconv.Itoa(hbm.round) + "_" + hbm.msgType + "_" + strconv.Itoa(hbm.channel) + "_" + hbm.msg
	fmt.Println("[sendMessages] Sending message ", finalMessage, " to ", to)
	_, err2 := writeFileHandlers[to].WriteString(finalMessage + "\n")

	if err2 != nil {
		log.Fatal(err2)
	}

	time.Sleep(500 * time.Millisecond)
}

func send(to int, msg hbMessage) {
	sendMessages(to, msg)
}

func broadcast(hbm hbMessage) {
	for i := 0; i < len(all_ports); i++ {
		sendMessages(i, hbm)
	}
}

func (hb *honeybadger) run_round(r int, txn string, hb_block chan []string, receiver *zmq.Socket) {
	fmt.Println("[run_round] Running round ", r, " proposed txn: ", txn)

	sid := hb.sid + ":" + strconv.Itoa(r)

	aba_inputs := make([](chan int), hb.N)
	aba_outputs := make([](chan int), hb.N)
	rbc_outputs := make([](chan string), hb.N)

	for j := 0; j < hb.N; j++ {
		aba_inputs[j] = make(chan int, 1)
		aba_outputs[j] = make(chan int, 1)
		rbc_outputs[j] = make(chan string, 1)
	}

	coin_recvs = make([](chan string), hb.N)
	aba_recvs = make([](chan string), hb.N)
	rbc_recvs = make([](chan string), hb.N)

	for j := 0; j < hb.N; j++ {
		coin_recvs[j] = make(chan string, 100)
		aba_recvs[j] = make(chan string, 100)
		rbc_recvs[j] = make(chan string, 100)
	}

	my_rbc_input := make(chan string, 1)

	setup := func(j int) {
		fmt.Println("[run_round] Setting up for node ", j)

		// These are supposed to be infinite sized channels. Initializing with
		// size N instead.

		fmt.Println("[run_round] Spawning binary agreement for node ", j)
		go binaryagreement(r, sid+"ABA"+strconv.Itoa(j), hb.pid, hb.N, hb.f, j, aba_inputs[j], aba_outputs[j], aba_recvs[j],
			sid+"COIN"+strconv.Itoa(j), hb.pid, hb.N, hb.f, hb.sPK, hb.sSK, coin_recvs[j])

		// These are supposed to be infinite sized channels. Initializing with
		// size N instead.

		fmt.Println("[run_round] Spawning reliable broadcast for node ", j)
		go reliablebroadcast(r, sid+"RBC"+strconv.Itoa(j), hb.pid, hb.N, hb.f, j, my_rbc_input, rbc_recvs[j], rbc_outputs[j])
	}

	for j := 0; j < hb.N; j++ {
		setup(j)
	}

	rbc_values := make([]chan string, hb.N)

	for i := 0; i < hb.N; i++ {
		rbc_values[i] = make(chan string, 1)
	}

	fmt.Println("[run_round] Spawning common subset")
	go commonsubset(hb.pid, hb.N, hb.f, rbc_outputs, aba_inputs, aba_outputs, rbc_values)

	fmt.Println("[run_round] Adding txn ", txn, " to input channel.")
	input := make(chan string, 1)
	input <- txn

	fmt.Println("[run_round] Calling honeybadger_block for round ", r)
	honeybadgerBlock(hb.pid, hb.N, hb.f, input, my_rbc_input, rbc_values, hb_block)

	fmt.Println("[run_round] *********** Round ", r, " is complete! ***********")
}

func (hb *honeybadger) run(receiver *zmq.Socket) {
	fmt.Println("Starting honeybadger")
	var new_txns []string
	hb_block := make(chan []string, 1)
	var proposed []string

	for round := 0; round < 3; round++ {
		start := time.Now()
		fmt.Println("[hooneybadger] round", round, " transaction_buffer: ", hb.transaction_buffer)
		currentRound = round
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

		fmt.Println("Sleeping for a while before going to next round.")
		t := time.Now()
		elapsed := t.Sub(start)

		fmt.Println("\n\n**********Time for round ", round, ": ", elapsed, "************\n\n")
		time.Sleep(5 * time.Second)
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

	// receiver.Bind("tcp://*:" + all_ports[pid])

	// go sync_nodes(receiver, pid)

	// time.Sleep(5 * time.Second)

	fmt.Println("[main] buffer: ", transaction_buffer, " N: ", N, " f: ", f, " B: ", B)

	var shares tcrsa.KeyShareList
	shares = make(tcrsa.KeyShareList, N)
	var meta *tcrsa.KeyMeta
	meta = &tcrsa.KeyMeta{}
	if _, err := os.Stat("./keys.txt"); err == nil {
		fmt.Println("[honeybadger] Loading keys from file")
		f, _ := os.Open("./keys.txt")
		defer f.Close()

		scanner := bufio.NewScanner(f)

		i := 0
		for scanner.Scan() {
			if i < N {
				json.Unmarshal([]byte(scanner.Text()), &shares[i])
			} else {
				err := json.Unmarshal([]byte(scanner.Text()), meta)
				if err != nil {
					fmt.Println("Loading public key error:", err)
				}
				continue
			}
			i++
		}

	} else if err != nil {
		shares, meta = threshsig.Dealer(4, 3, 2048)
	}

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

	if err := os.Truncate(files[pid], 0); err != nil {
		log.Printf("Failed to truncate: %v", err)
	}

	if err := os.Truncate("visualization/"+strconv.Itoa(pid)+".txt", 0); err != nil {
		log.Printf("Failed to truncate: %v", err)
	}

	for i := 0; i < N; i++ {
		fileLocks[i] = fslock.New(files[i])
		f, err := os.OpenFile(files[i], os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)

		if err != nil {
			log.Fatal(err)
		}

		writeFileHandlers[i] = f

		defer writeFileHandlers[i].Close()
	}

	readFileHandler, err = os.Open(files[pid])
	if err != nil {
		log.Fatal(err)
	}
	defer readFileHandler.Close()

	readScanner = bufio.NewScanner(readFileHandler)

	go func() {
		t, _ := tail.TailFile(files[pid], tail.Config{Follow: true})
		for line := range t.Lines {
			message := line.Text
			fmt.Println("[main] ", message)
			// Parse the message
			splitMsg := strings.Split(message, "_")
			round, _ := strconv.Atoi(splitMsg[0])

			if round == currentRound {
				msgType := splitMsg[1]
				channel, _ := strconv.Atoi(splitMsg[2])
				msg := strings.Join(splitMsg[3:], "_")

				if msgType != "IGNORE" {
					channel := getChannelFromMsg(msgType, channel)

					// Put message in apt channel
					channel <- msg
				}
			}
		}
	}()

	fmt.Println("[main] Ready to run honeybadger ", hb.N)
	hb.run(receiver)
}
