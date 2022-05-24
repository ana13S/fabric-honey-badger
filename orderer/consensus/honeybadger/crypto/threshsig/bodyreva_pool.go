package threshsig

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"fmt"

	tcrsa "github.com/niclabs/tcrsa"
)

var k uint16

const hashType = crypto.SHA256

// n is number of parties, k = id
func Dealer(N int, K int, size int) (shares tcrsa.KeyShareList, meta *tcrsa.KeyMeta) {
	n := uint16(N)
	k = uint16(K)

	// Generate keys
	keyShares, keyMeta, err := tcrsa.NewKey(size, k, n, nil)
	if err != nil {
		panic(fmt.Sprintf("%v", err))
	}

	return keyShares, keyMeta
}

func HashMessage(msg string, keyMeta *tcrsa.KeyMeta) ([]byte, []byte) {
	docHash := sha256.Sum256([]byte(msg))
	docPKCS1, err := tcrsa.PrepareDocumentHash(keyMeta.PublicKey.Size(), crypto.SHA256, docHash[:])
	if err != nil {
		panic(fmt.Sprintf("%v", err))
	}
	return docHash[:], docPKCS1
}

func Sign(keyShare tcrsa.KeyShare, docPK []byte, meta *tcrsa.KeyMeta) tcrsa.SigShare {
	sigShare, err := keyShare.Sign(docPK, hashType, meta)
	if err != nil {
		panic(fmt.Sprintf("%v", err))
	}
	return *sigShare
}

func GroupSign(K int, docPK []byte, keyShares tcrsa.KeyShareList, meta *tcrsa.KeyMeta) tcrsa.SigShareList {
	sigShares := make(tcrsa.SigShareList, k)
	var i uint16
	var err error

	// Sign with each node
	for i = 0; i < k; i++ {
		sigShares[i], err = keyShares[i].Sign(docPK, hashType, meta)
		if err != nil {
			panic(fmt.Sprintf("%v", err))
		}
		if err := sigShares[i].Verify(docPK, meta); err != nil {
			panic(fmt.Sprintf("%v", err))
		}
	}
	return sigShares
}

func CombineSignatures(docPK []byte, sigShares tcrsa.SigShareList, meta *tcrsa.KeyMeta) tcrsa.Signature {
	// Combine to create a real signature.
	signature, err := sigShares.Join(docPK, meta)
	if err != nil {
		panic(fmt.Sprintf("%v", err))
	}

	return signature
}

func Verify(meta *tcrsa.KeyMeta, docHash []byte, signature tcrsa.Signature) {
	// Check signature
	if err := rsa.VerifyPKCS1v15(meta.PublicKey, crypto.SHA256, docHash[:], signature); err != nil {
		panic(fmt.Sprintf("%v", err))
	}
}
