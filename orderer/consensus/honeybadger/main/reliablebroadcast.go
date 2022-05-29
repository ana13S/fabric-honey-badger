package main

import (
	"bytes"
	"crypto/sha256"

	//"fmt"

	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/klauspost/reedsolomon"
)

// clone of HBFT encode
func encode(K int, N int, m string) [][]byte {

	// pad m to a number of characters that is a multiple of K
	// DANGER: we assume that len(m) >= K
	if len(m)%K != 0 {
		padlen := K - (len(m) % K)
		m += strings.Repeat(string(rune(K-padlen)), padlen)
	}

	// convert m to byte slice
	mByteSlice := []byte(m)

	// initialize data container
	data := make([][]byte, N)

	// populate data container
	shardSize := len(m) / K
	for i := 0; i < N; i++ {

		data[i] = make([]byte, shardSize)

		if i < K {
			for j := 0; j < shardSize; j++ {
				data[i][j] = mByteSlice[shardSize*i+j]
			}
		}
	}

	//encode data container
	enc, _ := reedsolomon.New(K, N-K)
	_ = enc.Encode(data)

	return data

}

// clone of HBFT decode
func decode(K int, N int, data [][]byte) string {

	// localize data
	localData := data

	//decode data container
	dec, _ := reedsolomon.New(K, N-K)
	_ = dec.Reconstruct(localData)

	//recover padded string
	paddedString := ""
	for i := 0; i < K; i++ {
		paddedString += string(localData[i])
	}

	// remove padding
	paddedStringLen := len(paddedString)
	padlen := int(paddedString[paddedStringLen-1])
	trimmedString := paddedString[:paddedStringLen-padlen-1]

	return trimmedString

}

func hashLister(codeword [][]byte) [][]byte {
	var output [][]byte
	for i := 0; i < len(codeword); i++ {
		valHasher := sha256.New()
		valHasher.Write(codeword[i])
		outHash := valHasher.Sum(nil)
		output = append(output, outHash)
	}
	return output
}

func hashVerify(hashlist [][]byte, pos int, symbol []byte) bool {
	hasher := sha256.New()
	hasher.Write(symbol)
	symbolHash := hasher.Sum(nil)
	return bytes.Equal(symbolHash, hashlist[pos])
}

type Rb_msg struct {
	Pid      int
	Sid      int
	MsgType  string
	Roothash []byte
	Branch   [][]byte
	Stripe   []byte
}

func rb_msg_stringify(leader int, msg Rb_msg) hbMessage {
	marsh, _ := json.Marshal(msg)
	message := string(marsh)
	msgType := "RBC"
	sender := leader
	return hbMessage{
		msgType: msgType,
		sender:  sender,
		msg:     message,
	}
}

func rb_msg_parse(msg string) Rb_msg {
	var new_rb_msg Rb_msg
	json.Unmarshal([]byte(msg), &new_rb_msg)
	return new_rb_msg
}

func reliablebroadcast(
	sid string,
	pid int,
	N int,
	f int,
	leader int,
	input <-chan string,
	receive <-chan string,
	// send func(i int, msg string),
	retChan chan<- string) string {

	// default code
	K := N - 2*f            // Need this many to reconstruct. (# noqa: E221)
	EchoThreshold := N - f  // Wait for this many ECHO to send READY. (# noqa: E221)
	ReadyThreshold := f + 1 // Wait for this many READY to amplify READY. (# noqa: E221)
	OutputThreshold := 2*f + 1

	var m string
	if pid == leader {
		m = <-input

		// stripes is [][]byte
		codeword := encode(K, N, m)

		// use list hash
		hasher := sha256.New()
		hasher.Write([]byte(m))
		roothash := hasher.Sum(nil)
		branch := hashLister(codeword)

		for i := 0; i < N; i++ {

			toSend := rb_msg_stringify(
				leader,
				Rb_msg{
					Pid:      pid,
					Sid:      i,
					MsgType:  "VAL",
					Roothash: roothash,
					Branch:   branch,
					Stripe:   codeword[i],
				})

			time.Sleep(time.Second * 3)
			send(i, toSend)
		}
	}

	stripes := make(map[string]map[int][]byte)

	echoCounter := make(map[string]int)

	readySent := false

	ready := make(map[string]int)

	decode_output := func(roothash []byte) string {

		assembledStripes := [][]byte{}
		for i := 0; i < N; i++ {
			assembledStripes = append(assembledStripes, stripes[string(roothash)][i])
		}

		m := decode(K, N, assembledStripes)

		hasher := sha256.New()
		hasher.Write([]byte(m))
		if bytes.Equal(roothash, hasher.Sum(nil)) {
			return m
		} else {
			return "FAILURE"
		}
	}

	var rb_msg_raw string

	for {

		// read in marshaled string
		rb_msg_raw = <-receive

		// unmarshal
		recvd_msg := rb_msg_parse(rb_msg_raw)
		sender := recvd_msg.Pid
		roothash := recvd_msg.Roothash
		branch := recvd_msg.Branch
		stripe := recvd_msg.Stripe
		pos := recvd_msg.Sid

		// Verify received message
		if !hashVerify(branch, pos, stripe) {
			log.Println("Failed to validate message!")
			continue
		} else {
			log.Println("Message validated!")
		}

		if recvd_msg.MsgType == "VAL" {

			log.Println("Received val for roothash", roothash)

			if sender != leader {
				log.Println("VAL message from other than leader:", sender)
			}

			toBroadcast := rb_msg_stringify(
				leader,
				Rb_msg{
					Pid:      pid,
					Sid:      pos,
					MsgType:  "ECHO",
					Roothash: roothash,
					Branch:   branch,
					Stripe:   stripe,
				})

			for i := 0; i < N; i++ {

				time.Sleep(time.Second * 3)
				send(i, toBroadcast)
			}

		} else if recvd_msg.MsgType == "ECHO" {

			log.Println("Received echo for roothash", roothash)

			// update records
			toStripe := map[int][]byte{sender: stripe}
			stripes[string(roothash)] = toStripe
			log.Println(echoCounter)
			log.Println(sid, pid, N, leader)
			echoCounter[string(roothash)] += 1
			log.Println("+++++++++++++++++++++++++")
			log.Println("+++++++++++++++++++++++++")
			log.Println("Roothash", roothash, "has current echo count", echoCounter[string(roothash)])
			log.Println("+++++++++++++++++++++++++")
			log.Println("+++++++++++++++++++++++++")

			if echoCounter[string(roothash)] >= EchoThreshold && !readySent {
				readySent = true
				toBroadcast := rb_msg_stringify(
					leader,
					Rb_msg{
						Pid:      pid,
						Sid:      pos,
						MsgType:  "READY",
						Roothash: roothash,
						Branch:   branch,
						Stripe:   stripe,
					})
				for i := 0; i < N; i++ {

					time.Sleep(time.Second * 3)
					send(i, toBroadcast)
				}
			}

			headcount := ready[string(roothash)]
			echoCount := echoCounter[string(roothash)]
			if headcount >= OutputThreshold && echoCount >= K {
				log.Println("REACHED!!!!")
				retChan <- decode_output(roothash)
			}

		} else if recvd_msg.MsgType == "READY" {
			roothash := recvd_msg.Roothash
			ready[string(roothash)] += 1

			if ready[string(roothash)] >= ReadyThreshold && !readySent {
				readySent = true
				toBroadcast := rb_msg_stringify(
					leader,
					Rb_msg{
						Pid:      pid,
						Sid:      pos,
						MsgType:  "READY",
						Roothash: roothash,
						Branch:   branch,
						Stripe:   stripe,
					})
				for i := 0; i < N; i++ {

					time.Sleep(time.Second * 3)
					send(i, toBroadcast)
				}
			}

			if ready[string(roothash)] >= OutputThreshold && echoCounter[string(roothash)] >= K {
				log.Println("REACHED!!!!")
				retChan <- decode_output(roothash)
			}
		} else {
			log.Println("ERROR: Message Type Unknown!!!")
		}

	}

}
