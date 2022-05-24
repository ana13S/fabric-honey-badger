package main

import (
	"bytes"
	"crypto/sha256"
	// "fmt"
	"log"
	"strings"

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

/*
// clone HBFT hash
// note we assume input is byte slice or a string
func hash(x interface{}) string {

	// create hash object
	h := sha256.New()

	// switch depending on whether x is byte slice or string
	switch x.(type) {

	case string:

		xBytes := []byte(fmt.Sprintf("%v", x))
		h.Write(xBytes)

	case []byte:

		xBytes := x.([]byte)
		h.Write(xBytes)

	}

	// return hex string
	return fmt.Sprintf("%x", h.Sum(nil))

}

// clone HBFT ceil
func ceil(x float64) float64 {
	return math.Ceil(x)
}
*/

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

// Channel cannot return tuple so I defined a struct for receive/send
type rb_msg struct {
	pid   int
	iface []interface{}
}

func reliablebroadcast(sid string, pid int, N int, f int, leader int, input <-chan string, receive <-chan rb_msg, send func(int, []interface{}), retChan chan<- string) string {

	// default code
	K := N - 2*f            // Need this many to reconstruct. (# noqa: E221)
	EchoThreshold := N - f  // Wait for this many ECHO to send READY. (# noqa: E221)
	ReadyThreshold := f + 1 // Wait for this many READY to amplify READY. (# noqa: E221)
	OutputThreshold := 2*f + 1

	broadcast := func(o []interface{}) {
		for i := 0; i < N; i++ {
			send(i, o)
		}
	}

	var m string
	if pid == leader {
		m = <-input

		stripes := encode(K, N, m)

		mt := makeMerkleTree(stripes)

		roothash := mt.MerkleRoot()

		for i := 0; i < N; i++ {
			branch := getMerkleBranch(stripes[i], mt)
			toSend := []interface{}{"VAL", roothash, branch, stripes[i]}
			send(i, toSend)
		}
	}

	var fromLeader []byte

	stripes := make(map[string][][]byte)

	echoCounter := make(map[string]int)

	readySent := false
	echoSenders := mapset.NewSet()
	readySenders := mapset.NewSet()

	ready := make(map[string]mapset.Set)

	decode_output := func(roothash []byte) string {
		m := decode(K, N, stripes[string(roothash)])
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
		rb_msg_received := <-receive
		msg := rb_msg_received.iface
		sender := rb_msg_received.pid

		if msg[0] == "VAL" && fromLeader == nil {

			roothash := msg[1].([]byte)
			branch := msg[2].([][]byte)
			stripe := msg[3].([]byte)

			if sender != leader {
				log.Println("VAL message from other than leader:", sender)
			}

			if !merkleVerify(stripe, roothash, branch) {
				log.Println("Failed to validate ECHO message!")
			}
			// need to add error handling

			fromLeader = roothash
			toBroadcast := []interface{}{"ECHO", roothash, branch, stripe}
			broadcast(toBroadcast)

		} else if msg[0] == "ECHO" {

			roothash := msg[1].([]byte)
			branch := msg[2].([][]byte)
			stripe := msg[3].([]byte)

			if !merkleVerify(stripe, roothash, branch) {
				log.Println("Failed to validate ECHO message!")
			}
			// need to add error handling

			// update
			stripes[string(roothash)][sender] = stripe
			echoSenders.Add(sender)
			echoCounter[string(roothash)] += 1

			if echoCounter[string(roothash)] >= EchoThreshold && !readySent {
				readySent = true
				toBroadcast := []interface{}{"READY", roothash}
				broadcast(toBroadcast)
			}

			if ready[string(roothash)].Cardinality() >= OutputThreshold && echoCounter[string(roothash)] >= K {
				retChan <- decode_output(roothash)
			}

		} else if msg[0] == "READY" {
			roothash := msg[1].([]byte)
			ready[string(roothash)].Add(sender)
			readySenders.Add(sender)

			if ready[string(roothash)].Cardinality() >= ReadyThreshold && !readySent {
				readySent = true
				toBroadcast := []interface{}{"READY", roothash}
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
		fmt.Println(merkleTree(codeword)[1])


}*/
