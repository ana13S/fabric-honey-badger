package main

import (
	// "context"
	// "crypto/rand"
	// "crypto/tls"
	"fmt"
	"os"
	// "github.com/apache/thrift/lib/go/thrift"
	// "github.com/stretchr/testify/assert"
	// "github.com/vishalmohanty/encryption"
	// "github.com/vishalmohanty/go_threshenc"
)

/*
   The HoneyBadgerBFT algorithm for a single block

   :param pid: my identifier
   :param N: number of nodes
   :param f: fault tolerance
   :param PK: threshold encryption public key
   :param SK: threshold encryption secret key
   :param propose_in: a channel returning a sequence of transactions
   :param acs_in: a channel to provide input to acs routine
   :param acs_out: a blocking channel that returns an array of ciphertexts
   :param hp_block: a channel that returns a dictionary of shares received
   :param tpke_bcast:
   :param tpke_recv:
   :return: slice of strings (txns) committed in this round
*/
func honeybadgerBlock(pid int, N int, f int, propose_in <-chan string, acs_in <-chan []string, acs_out <-chan []string, hb_block chan<- []string) {

	// Broadcast inputs are of the form (tenc(key), enc(key, transactions))

	prop := <-propose_in

	acs_in <- prop

	// Wait for the corresponding ACS to finish
	vall := <-acs_out
	if len(vall) != N {
		fmt.Println("len(vall): ", len(vall), " N: ", N, " are not equal.")
		os.Exit(1)
	}
	var nonNil = 0
	for _, val := range vall { // This many must succeed
		if val != nil {
			nonNil = nonNil + 1
		}
	}
	if nonNil < (N - f) {
		fmt.Println("non nil values: ", nonNil, " should be at least: ", N-f)
		os.Exit(1)
	}

	hb_block <- vall
}
