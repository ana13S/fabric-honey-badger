// Based on https://github.com/initc3/HoneyBadgerBFT-Python/blob/dev/test/crypto/threshsig/test_boldyreva.py
package crypto

import (
	"fmt"
	"testing"
)

// Don't need to use goroutine or multithreading for this test
// If we need single threads signing, we can use tcrsa.Sign (which is used in combine in this case)

func Test_boldyreva(t *testing.T) {
	shares, meta, _ := dealer(5, 3, 2048)
	docHash, docPK := hash_message("hi", meta)
	fmt.Print("Hashed message: ")
	fmt.Print(docHash)
	signature := combine(3, docPK, shares, meta)
	fmt.Print("\nSignature :")
	fmt.Print(signature)
	verify(meta, docHash, signature)
	fmt.Print("\nverified")
}
