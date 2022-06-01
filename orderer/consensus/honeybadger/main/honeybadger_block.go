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
func honeybadgerBlock(pid int, N int, f int, propose_in <-chan string, acs_in chan<- string, acs_out []chan string, hb_block chan<- []string) {

	// Broadcast inputs are of the form (tenc(key), enc(key, transactions))

	fmt.Println("[honeybadger_block] Entering now")

	prop := <-propose_in

	acs_in <- prop

	// Wait for the corresponding ACS to finish
	var vall = make([]string, N)
	for i, val := range acs_out {
		vall[i] = <-val
	}
	if len(vall) != N {
		fmt.Println("len(vall): ", len(vall), " N: ", N, " are not equal.")
		os.Exit(1)
	}
	var nonNil = 0
	for _, val := range vall { // This many must succeed
		if val != "" {
			nonNil = nonNil + 1
		}
	}
	if nonNil < (N - f) {
		fmt.Println("non nil values: ", nonNil, " should be at least: ", N-f)
		os.Exit(1)
	}

	var committed_txns []string
	txn_map := make(map[string]bool)
	for _, txn := range vall {
		if txn != "FAILURE" {
			_, ok := txn_map[txn]
			if !ok {
				committed_txns = append(committed_txns, txn)
				txn_map[txn] = true
			}
		}
	}

	fmt.Println("[honeybadger_block] Committed Transactions", committed_txns)

	hb_block <- committed_txns
}
