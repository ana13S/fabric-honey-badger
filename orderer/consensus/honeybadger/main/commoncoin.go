package main

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"fmt"

	tcrsa "github.com/niclabs/tcrsa"
)

func hash(msg string) {
	h:= sha256.New()
	h.Write(str.encode(msg))
	return h.Sum(nil)
}

// keyMeta holds public key 
// keyShare holds values for specific node(including something similar to secret key)
func shared_coin(sid int, pid int, N int, f int, meta tcrsa.KeyMeta, keyShare tcrsa.KeyShare, 
	broadcast func(int), receive chan string, getCoinFunc <-chan (func(int) int)) {
	if meta.L != N || meta.L != f + 1: // assert PK.l == N   assert PK.k == f+1 
		panic("F and N not set correctly")
	

	// Need to get r from receive
	docHash, docPK := hash_message(sid + r, meta) // h = PK.hash_message(str((sid, r)))

	// For now, don't need to verify each share. There is no easy equivalent method in tcrsa at the moment

    // Calculate signature to give to others
    sigShare = sign(keyShare, docPK, meta)

	// After receiving signatures from others

	meta.L = f + 1 // Need f +1 keyshares to verify signature
	shares := make(tcrsa.SigShareList, meta.L)
	for i = 0: i < f + 1; i++ {
		sigShares[i] = sigShare
	}


	var sig string
	sig = combine_signatures(docPK, sigShares, meta)//  sig = PK.combine_shares(sigs) assert PK.verify_signature(sig, h)
	verify(meta, docHash, sig) // This will trigger a panic if verification fails

	bit := hash(signature)[0]%2 // bit = hash(serialize(sig))[0] % 2


	// In getCoin()'s initialization
	sigShare = sign(keyShare, docPK, meta) // SK.sign(h)
	sigShare.Xi // This is signature share value. Can be combined with f other signatures to verify a msg
	sigShare.C // Verification value of signature share

}

/**
    """A shared coin based on threshold signatures

    :param sid: a unique instance id
    :param pid: my id number
    :param N: number of parties
    :param f: fault tolerance, :math:`f+1` shares needed to get the coin
    :param PK: ``boldyreva.TBLSPublicKey``
    :param SK: ``boldyreva.TBLSPrivateKey``
    :param broadcast: broadcast channel
    :param receive: receive channel
    :return: a function ``getCoin()``, where ``getCoin(r)`` blocks
    """
    assert PK.k == f+1
    assert PK.l == N    # noqa: E741
    received = defaultdict(dict)
    outputQueue = defaultdict(lambda: Queue(1))

    def _recv():
        while True:     # main receive loop
            logger.debug(f'entering loop',
                         extra={'nodeid': pid, 'epoch': '?'})
            # New shares for some round r, from sender i
            (i, (_, r, sig)) = receive()
            logger.debug(f'received i, _, r, sig: {i, _, r, sig}',
                         extra={'nodeid': pid, 'epoch': r})
            assert i in range(N)
            assert r >= 0
            if i in received[r]:
                print("redundant coin sig received", (sid, pid, i, r))
                continue

            h = PK.hash_message(str((sid, r)))

            # TODO: Accountability: Optimistically skip verifying // Don't need to do this for now
            # each share, knowing evidence available later
            try:
                PK.verify_share(sig, i, h)
            except AssertionError:
                print("Signature share failed!", (sid, pid, i, r))
                continue

            received[r][i] = sig

            # After reaching the threshold, compute the output and
            # make it available locally
            logger.debug(
                f'if len(received[r]) == f + 1: {len(received[r]) == f + 1}',
                extra={'nodeid': pid, 'epoch': r},
            )
            if len(received[r]) == f + 1:

                # Verify and get the combined signature
                sigs = dict(list(received[r].items())[:f+1])
                sig = PK.combine_shares(sigs)
                assert PK.verify_signature(sig, h)

                # Compute the bit from the least bit of the hash
                bit = hash(serialize(sig))[0] % 2
                logger.debug(f'put bit {bit} in output queue',
                             extra={'nodeid': pid, 'epoch': r})
                outputQueue[r].put_nowait(bit)

    # greenletPacker(Greenlet(_recv), 'shared_coin', (pid, N, f, broadcast, receive)).start()
    Greenlet(_recv).start()

    def getCoin(round):
        """Gets a coin.

        :param round: the epoch/round.
        :returns: a coin.

        """
        # I have to do mapping to 1..l
        h = PK.hash_message(str((sid, round)))
        logger.debug(f"broadcast {('COIN', round, SK.sign(h))}",
                     extra={'nodeid': pid, 'epoch': round})
        broadcast(('COIN', round, SK.sign(h)))
        return outputQueue[round].get()

    return getCoin
**/ 