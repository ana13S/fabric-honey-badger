package main

import (
	"bytes"
	"crypto/sha256"

	//"fmt"

	"log"
	"strings"

	"encoding/json"

	"github.com/cbergoon/merkletree"

	mapset "github.com/deckarep/golang-set"
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

//TestContent implements the Content interface provided by merkletree and represents the content stored in the tree.
type TestContent struct {
	x []byte
}

//CalculateHash hashes the values of a TestContent
func (t TestContent) CalculateHash() ([]byte, error) {
	h := sha256.New()
	if _, err := h.Write(t.x); err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}

//Equals tests for equality of two Contents
func (t TestContent) Equals(other merkletree.Content) (bool, error) {

	res := bytes.Compare(t.x, other.(TestContent).x)
	if res == 0 {
		return true, nil
	} else {
		return false, nil
	}
}

func makeMerkleTree(shardData [][]byte) *merkletree.MerkleTree {

	// convert shardData to TestContent
	var list []merkletree.Content
	for _, shard := range shardData {
		list = append(list, TestContent{x: shard})
	}

	t, _ := merkletree.NewTree(list)

	return t

}

func getMerkleBranch(val []byte, mt *merkletree.MerkleTree) [][]byte {

	branch, _, _ := mt.GetMerklePath(TestContent{x: val})

	return branch

}

func merkleVerify(val []byte, roothash []byte, branch [][]byte) bool {

	valHasher := sha256.New()
	valHasher.Write(val)
	runningHash := valHasher.Sum(nil)

	for _, branchHash := range branch {

		h := sha256.New()
		h.Write(runningHash)
		h.Write(branchHash)
		runningHash = h.Sum(nil)

	}

	if bytes.Equal(runningHash, roothash) {
		return true
	} else {
		return false
	}

}

type Rb_msg struct {
	Pid      int
	Msg_type string
	Roothash []byte
	Branch   [][]byte
	Stripe   []byte
}

func rb_msg_stringify(pid int, msg Rb_msg) hbMessage {
	marsh, _ := json.Marshal(msg)
	message := string(marsh)
	msgType := "RBC"
	sender := pid
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

	// broadcast := func(msg string) {
	// 	for i := 0; i < N; i++ {
	// 		send(i, msg)
	// 	}
	// }

	var m string
	if pid == leader {
		m = <-input

		// stripes is [][]byte
		stripes := encode(K, N, m)

		// pointer to a local merkletree.MerkleTree object
		mt := makeMerkleTree(stripes)

		// roothash is []byte
		roothash := mt.MerkleRoot()

		for i := 0; i < N; i++ {
			branch := getMerkleBranch(stripes[i], mt)

			toSend := rb_msg_stringify(
				pid,
				Rb_msg{
					Pid:      pid,
					Msg_type: "VAL",
					Roothash: roothash,
					Branch:   branch,
					Stripe:   stripes[i],
				})

			send(i, toSend)
		}
	}

	stripes := make(map[string]map[int][]byte)

	echoCounter := make(map[string]int)

	readySent := false
	echoSenders := mapset.NewSet()
	readySenders := mapset.NewSet()

	ready := make(map[string]mapset.Set)

	decode_output := func(roothash []byte) string {

		assembledStripes := [][]byte{}
		for i := 0; i < N; i++ {
			assembledStripes = append(assembledStripes, stripes[string(roothash)][i])
		}

		m := decode(K, N, assembledStripes)
		_stripes := encode(K, N, m)
		_mt := makeMerkleTree(_stripes)
		_roothash := _mt.MerkleRoot()
		if bytes.Equal(_roothash, roothash) {
			return m
		} else {
			return "FAILURE"
		}
	}

	for {

		// read in marshaled string
		rb_msg_raw := <-receive

		// unmarshal
		recvd_msg := rb_msg_parse(rb_msg_raw)
		sender := recvd_msg.Pid

		if recvd_msg.Msg_type == "VAL" {

			roothash := recvd_msg.Roothash
			branch := recvd_msg.Branch
			stripe := recvd_msg.Stripe

			if sender != leader {
				log.Println("VAL message from other than leader:", sender)
			}

			if !merkleVerify(stripe, roothash, branch) {
				log.Println("Failed to validate ECHO message!")
				continue
			}

			toBroadcast := rb_msg_stringify(
				pid,
				Rb_msg{
					Pid:      pid,
					Msg_type: "ECHO",
					Roothash: roothash,
					Branch:   branch,
					Stripe:   stripe,
				})
			broadcast(toBroadcast)

		} else if recvd_msg.Msg_type == "ECHO" {

			roothash := recvd_msg.Roothash
			branch := recvd_msg.Branch
			stripe := recvd_msg.Stripe

			if !merkleVerify(stripe, roothash, branch) {
				log.Println("Failed to validate ECHO message!")
				continue
			}

			// update records
			toStripe := map[int][]byte{sender: stripe}
			stripes[string(roothash)] = toStripe
			echoSenders.Add(sender)
			echoCounter[string(roothash)] += 1

			if echoCounter[string(roothash)] >= EchoThreshold && !readySent {
				readySent = true
				toBroadcast := rb_msg_stringify(
					pid,
					Rb_msg{
						Pid:      pid,
						Msg_type: "READY",
						Roothash: roothash,
						Branch:   nil,
						Stripe:   nil,
					})
				broadcast(toBroadcast)
			}

			if ready[string(roothash)].Cardinality() >= OutputThreshold && echoCounter[string(roothash)] >= K {
				retChan <- decode_output(roothash)
			}

		} else if recvd_msg.Msg_type == "READY" {
			roothash := recvd_msg.Roothash
			ready[string(roothash)].Add(sender)
			readySenders.Add(sender)

			if ready[string(roothash)].Cardinality() >= ReadyThreshold && !readySent {
				readySent = true
				toBroadcast := rb_msg_stringify(
					pid,
					Rb_msg{
						Pid:      pid,
						Msg_type: "READY",
						Roothash: roothash,
						Branch:   nil,
						Stripe:   nil,
					})
				broadcast(toBroadcast)
			}

			if ready[string(roothash)].Cardinality() >= OutputThreshold && echoCounter[string(roothash)] >= K {
				retChan <- decode_output(roothash)
			}
		}

	}

}

/*
func main() {

	/*
		// Erasure Code Testing

			codeword := encode(5, 20, "helloworldys")
			fmt.Println(codeword)
			for i := 0; i < 15; i++ {
				codeword[i] = nil
			}
			fmt.Println(codeword)
			recoveredString := decode(5, 20, codeword)
			fmt.Println(recoveredString)
			fmt.Println(len(recoveredString))
*/

// Hash Testing
/*
	someHash := hash([]byte("Hello World"))
	fmt.Println(someHash)
	anotherHash := hash("Hello World")
	fmt.Println(anotherHash)
*/

// Ceil Testing
/*
	someFloat := 3.14
	fmt.Println(ceil(someFloat))
*/

// Merkle Tree Testing

/*
		codeword := encode(5, 20, "helloworldys")
		mt := makeMerkleTree(codeword)
		roothash := mt.MerkleRoot()
		branch := getMerkleBranch(codeword[0], mt)

		rbm := Rb_msg{
			Pid:      0,
			Msg_type: "Type",
			Roothash: roothash,
			Branch:   branch,
			Stripe:   codeword[0],
		}

		fmt.Println(rbm)

		rbm_strung := rb_msg_stringify(rbm)

		fmt.Println("HIWHAT")
		fmt.Println(rbm_strung)
		fmt.Println("HIWHAT")

		rbm_reloaded := rb_msg_parse(rbm_strung)

		fmt.Println(rbm_reloaded)
} */
