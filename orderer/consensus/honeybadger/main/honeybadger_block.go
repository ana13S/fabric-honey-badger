package main

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"os"

	"github.com/apache/thrift/lib/go/thrift"
	"github.com/vishalmohanty/go_threshenc"
	"github.com/vishalmohanty/encryption"
	"github.com/stretchr/testify/assert"
)

func honeybadgerBlock(pid int32, N int32, f int32, PK *go_threshenc.TPKEPublicKey, SK *go_threshenc.TPKEPrivateKey, 
    propose_in <-chan string, acs_in <-chan []string, acs_out <-chan []string, hb_block chan<- map[int32][]byte, 
    tpke_bcast, tpke_recv <-chan []string) {
    /**The HoneyBadgerBFT algorithm for a single block

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
    :return:
    **/

    // Broadcast inputs are of the form (tenc(key), enc(key, transactions))

    // Threshold encrypt
    // TODO: check that propose_in is the correct length, not too large
    prop := <- propose_in
    b := make([]byte, 32)
    key, err := rand.Read(b)
//     fmt.Println(n, err, b)
//     key = os.urandom(32)    // random 256-bit key
    aesKey := &go_threshenc.AESKey{key}
    ciphertext := aesKey.aesEncrypt(prop)
    tkey := PK.encrypt(key)

    //import pickle TODO
    //to_acs = pickle.dumps((serialize_UVW(*tkey), ciphertext))
    acs_in <- to_acs

    // Wait for the corresponding ACS to finish
    vall := <- acs_out 
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
        fmt.Println("non nil values: ", nonNil, " should be at least: ", N - f)
        os.Exit(1)
    }

    // print pid, 'Received from acs:', vall

    // Broadcast all our decryption shares
    my_shares := make([][]byte)
    for _, v in enumerate(vall):
        if v is nil:
            my_shares = append(my_shares, nil)
            continue
        (tkey, ciph) = pickle.loads(v)
//         tkey = deserialize_UVW(*tkey)
        share = SK.decrypt_share(*tkey)
        // share is of the form: U_i, an element of group1
        my_shares = append(my_shares, share)

    tpke_bcast(my_shares)

    // Receive everyone's shares
//     shares_received = {}
    shares_received := make(map[int32][]byte)
    sharesLen := 0
    for sharesLen < f+1 {
        (j, shares) = tpke_recv()
        if j in shares_received:
            // TODO: alert that we received a duplicate
            fmt.Println("Received a duplicate decryption share from ", j)
            continue
        shares_received[j] = shares
        sharesLen += 1
    }
    
    if len(shares_received) < f+1 {
        fmt.Println("Shares received: ", shares_received, ", should be at least ", f+1)
        os.Exit(1)
    }

    block <- shares_received

    // TODO: Accountability
    // If decryption fails at this point, we will have evidence of misbehavior,
    // but then we should wait for more decryption shares and try again
    decryptions := [][]byte
    for i, v in enumerate(vall):
        if v is nil:
            continue
        svec := make(map[int32][]byte)
        for j, shares := range shares_received:
            svec[j] = shares[i]     // Party j's share of broadcast i
        (tkey, ciph) = pickle.loads(v)
//         tkey = deserialize_UVW(*tkey)
        key = PK.combine_shares(*tkey, svec)
        plain = go_threshenc.aesDecrypt(key, ciph)
        decryptions = append(decryptions, plain)
    // print 'Done!', decryptions

    return decryptions
}