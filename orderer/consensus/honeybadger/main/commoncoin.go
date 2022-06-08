package main

import (
	"crypto/sha256"
	"encoding/json"
	// "fmt"
	"fmt"
	tcrsa "github.com/niclabs/tcrsa"
	"strconv"
	"strings"
	"threshsig"
	"time"
)

func hash(msg string) []byte {
	h := sha256.New()
	data := []byte(msg)
	h.Write(data)
	return h.Sum(nil)
}

func sig_msg_stringify(round int, pid int, r int, msg tcrsa.SigShare) hbMessage {
	marsh, _ := json.Marshal(msg)
	message := string(marsh)
	msgType := "COIN"
	channel := pid
	return hbMessage{
		round:   round,
		msgType: msgType,
		channel: channel,
		msg:     strconv.Itoa(r) + "_" + message,
	}
}

func sig_msg_parse(msg string) tcrsa.SigShare {
	var new_sig_msg tcrsa.SigShare
	json.Unmarshal([]byte(msg), &new_sig_msg)
	return new_sig_msg
}

func put_delay(msg string, receive chan string) {
	time.Sleep(time.Second * 1)
	receive <- msg
}

func broadcast_loop(msg hbMessage, killChan chan string) {
	for {
		select {
		case val := <-killChan:
			fmt.Println("[commoncoin] Finish broadcast because we received enough signatures" + val)
			return
		default:
			fmt.Println("Keep broadcasting signature")
			broadcast(msg)
		}
	}
}

// keyMeta holds public key
// keyShare holds values for specific node(including something similar to secret key)
func shared_coin(round int, sid string, pid int, N int, f int, channel int, meta tcrsa.KeyMeta, keyShare tcrsa.KeyShare,
	receive chan string, r int) int {

	docHash, docPK := threshsig.HashMessage(sid+strconv.Itoa(r), &meta) // h = PK.hash_message(str((sid, r)))

	// Calculate signature and broadcast to others
	sigShare := threshsig.Sign(keyShare, docPK, &meta)
	fmt.Println("[commoncoin] Generated signature. Need to broadcast")
	// killChan := make(chan string, 10)

	// Wait for K keyshares (N - f)
	shares := make([]tcrsa.SigShare, meta.K)
	shareList := make(tcrsa.SigShareList, meta.K)
	shareList[0] = &sigShare
	shareMap := make(map[uint16]bool)
	shareMap[sigShare.Id] = true

	// Keep broadcasting till we get enough sigShares
	broadcast(sig_msg_stringify(round, channel, r, sigShare))

	for i := 1; i < int(meta.K); {
		msg := <-receive
		received := strings.Split(msg, "_") // Get round of sigShare
		if received[0] != strconv.Itoa(r) {
			fmt.Println("[commoncoin] In round ", r, ", Received signature share from node in wrong round, ", received[0])
			go put_delay(msg, receive)
			continue
		}
		shares[i] = sig_msg_parse(received[1])
		_, ok := shareMap[shares[i].Id]
		if !ok {
			shareMap[shares[i].Id] = true
			shareList[i] = &shares[i]
			i++
			fmt.Println("[commoncoin] Received one signature share")
		}
	}

	fmt.Println("[commoncoin] Received signature share list ", shareList[0].Id, shareList[1].Id, shareList[2].Id)

	// After receiving signatures from others
	sig := threshsig.CombineSignatures(docPK, shareList, &meta)
	threshsig.Verify(&meta, docHash, sig)
	fmt.Println("[commoncoin] Success!! Signature list verified")
	bit := int(sig[0]) % 2
	return bit

}
