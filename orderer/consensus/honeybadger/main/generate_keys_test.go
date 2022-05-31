package main

import (
	"encoding/json"
	"log"
	"os"
	"testing"
	"threshsig"
)

func Test_generateKeys(t *testing.T) {
	shares, meta := threshsig.Dealer(4, 3, 2048)
	f, err := os.Create("keys.txt")

	if err != nil {
		log.Fatal("Could not create keys.txt")
	}

	sharesJson, _ := json.Marshal(shares)
	metaJson, _ := json.Marshal(meta)

	for _, s := range shares {
		share, _ := json.Marshal(s)
		f.Write(share)
		f.WriteString(string('\n'))
	}
	f.Write(sharesJson)
	f.WriteString(string('\n'))

	f.Write(metaJson)
	f.WriteString(string('\n'))

	defer f.Close()
}
