// Based on https://github.com/initc3/HoneyBadgerBFT-Python/blob/dev/test/crypto/threshsig/test_boldyreva.py
package threshsig

import (
	"testing"
)

// Don't need to use goroutine or multithreading for this test
// If we need single threads signing, we can use tcrsa.Sign (which is used in combine in this case)

func Test_boldyreva(t *testing.T) {
	shares, meta := Dealer(5, 3, 2048)
	docHash, docPK := HashMessage("hi", meta)
	//fmt.Print("Hashed message: ")
	//fmt.Print(docHash)
	sigShares := GroupSign(3, docPK, shares, meta) // Get group signature. In shared coin, we'll call sign
	signature := CombineSignatures(docPK, sigShares, meta)
	//fmt.Print("\nSignature :")
	//fmt.Print(signature)
	Verify(meta, docHash, signature)
	//fmt.Print("\nverified\n")
}
