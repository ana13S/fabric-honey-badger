package crypto

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"fmt"

	"./tcrsa"
)

var k int16 = 3 // Can change this in the future
var l int16 = 5

func generate_sigs(msg string, size int) {

	keyShares, keyMeta, err := tcrsa.NewKey(size, uint16(k), uint16(l), nil)
	if err != nil {
		panic(fmt.Sprintf("%v", err))
	}

	docHash := sha256.Sum256([]byte(msg))
	docPKCS1, err := tcrsa.PrepareDocumentHash(keyMeta.PublicKey.Size(), crypto.SHA256, docHash[:])
	if err != nil {
		panic(fmt.Sprintf("%v", err))
	}

	sigShares := make(tcrsa.SigShareList, l)
	return keyMeta, keyShares, sigShares, docPKCS1, docHash
}

func combine_and_verify(keyMeta KeyMeta, keyShares KeyShareList, sigShares SigShareList, docPK []byte, docHash []byte) {
	var i uint16

	for i = 0; i < l; i++ {
		sigShares[i], err = keyShares[i].Sign(docPK, crypto.SHA256, keyMeta)
		if err != nil {
			panic(fmt.Sprintf("%v", err))
		}
		if err := sigShares[i].Verify(docPKCS1, keyMeta); err != nil {
			panic(fmt.Sprintf("%v", err))
		}
	}

	signature, err := sigShares.Join(docPK, keyMeta)
	if err != nil {
		panic(fmt.Sprintf("%v", err))
	}

	if err := rsa.VerifyPKCS1v15(keyMeta.PublicKey, crypto.SHA256, docHash[:], signature); err != nil {
		panic(fmt.Sprintf("%v", err))
	}

}
