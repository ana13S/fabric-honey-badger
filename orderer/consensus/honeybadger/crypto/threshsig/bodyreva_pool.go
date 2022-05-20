package crypto

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
func dealer(N int, K int, size int) (shares tcrsa.KeyShareList, meta *tcrsa.KeyMeta, err error) {
	n := uint16(N)
	k = uint16(K)

	// Generate keys
	keyShares, keyMeta, err := tcrsa.NewKey(size, k, n, nil)
	if err != nil {
		panic(fmt.Sprintf("%v", err))
	}

	return keyShares, keyMeta, err
}

func hash_message(msg string, keyMeta *tcrsa.KeyMeta) ([]byte, []byte) {
	docHash := sha256.Sum256([]byte(msg))
	docPKCS1, err := tcrsa.PrepareDocumentHash(keyMeta.PublicKey.Size(), crypto.SHA256, docHash[:])
	if err != nil {
		panic(fmt.Sprintf("%v", err))
	}
	return docHash[:], docPKCS1
}

func combine(K int, docPK []byte, keyShares tcrsa.KeyShareList, meta *tcrsa.KeyMeta) tcrsa.Signature {
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

	// Combine to create a real signature.
	signature, err := sigShares.Join(docPK, meta)
	if err != nil {
		panic(fmt.Sprintf("%v", err))
	}

	return signature
}

func verify(meta *tcrsa.KeyMeta, docHash []byte, signature tcrsa.Signature) {
	// Check signature
	if err := rsa.VerifyPKCS1v15(meta.PublicKey, crypto.SHA256, docHash[:], signature); err != nil {
		panic(fmt.Sprintf("%v", err))
	}
}
