package main

import (
	"crypto/sha256"
	"encoding/json"
	// "fmt"
	tcrsa "github.com/niclabs/tcrsa"
	"strconv"
	"threshsig"
)

func hash(msg string) []byte {
	h := sha256.New()
	data := []byte(msg)
	h.Write(data)
	return h.Sum(nil)
}

func sig_msg_stringify(pid int, msg tcrsa.SigShare) hbMessage {
	marsh, _ := json.Marshal(msg)
	message := string(marsh)
	msgType := "COIN"
	sender := pid
	return hbMessage{
		msgType: msgType,
		sender:  sender,
		msg:     message,
	}
}

func sig_msg_parse(msg string) tcrsa.SigShare {
	var new_sig_msg tcrsa.SigShare
	json.Unmarshal([]byte(msg), &new_sig_msg)
	return new_sig_msg
}

// keyMeta holds public key
// keyShare holds values for specific node(including something similar to secret key)
func shared_coin(sid string, pid int, N int, f int, meta tcrsa.KeyMeta, keyShare tcrsa.KeyShare,
	receive chan string, r int) int {
	if int(meta.L) != N || int(meta.L) != f+1 { // assert PK.l == N   assert PK.k == f+1
		panic("F and N not set correctly")
	}

	// Need to get r from receive
	docHash, docPK := threshsig.HashMessage(sid+strconv.Itoa(r), &meta) // h = PK.hash_message(str((sid, r)))

	// Calculate signature and broadcast to others
	sigShare := threshsig.Sign(keyShare, docPK, &meta)
	broadcast(sig_msg_stringify(pid, sigShare))

	// Wait for f+1 keyshares
	meta.L = uint16(f + 1) // Need f +1 keyshares to verify signature
	shares := make([]tcrsa.SigShare, meta.L)
	shareList := make(tcrsa.SigShareList, meta.L)
	for i := 0; i < f+1; i++ {
		msg := <-receive
		shares[i] = sig_msg_parse(msg)
		shareList[i] = &shares[i]
	}

	// After receiving signatures from others
	sig := threshsig.CombineSignatures(docPK, shareList, &meta)
	threshsig.Verify(&meta, docHash, sig)
	bit := int(sig[0]) % 2
	return bit

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