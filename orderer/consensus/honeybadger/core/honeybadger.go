package core

import (
    "fmt"
    "time"
	"core/shared_coin"
	"core/binaryagreement"
	"core/reliablebroadcast"
	"core/commonsubset"
	"core/honeybadger_block"
)

type BroadcastTag struct {
	name str;
	ACS_COIN str;
	ACS_ABA str;
	TPKE str;
}



var BroadcastReceiverQueues = BroadcastTag("BroadcastReceiverQueues", "ACS_COIN", "ACS_ABA", "ACS_RBC", "TPKE")


// TODO convert interfaces to function with proper types defined.
func broadcast_receiver(recv_func interface {}, recv_queues interface {
}) {
	var j interface {
	}
	var tag interface {
	}
	sender, [3]interface {
	}{tag, j, msg} := recv_func()
	if func() int {
		for i, v := range BroadcastTag.__members__ {
			if v == tag {
				return i
			}
		}
		return -1
	}() == -1 {
		panic(fmt.Errorf("UnknownTagError: %v", "Unknown tag: {}! Must be one of {}.".format(tag, BroadcastTag.__members__)))
	}
	recv_queue := recv_queues._asdict()[tag]
	if tag != BroadcastTag.TPKE.value {
		recv_queue = recv_queue[j]
	}
	recv_queue.put_nowait([2]interface {
	}{sender, msg})
}

func broadcast_receiver_loop(recv_func interface {
}, recv_queues interface {
}) {
	for {
		broadcast_receiver(recv_func, recv_queues)
	}
}
package main

type HoneyBadgerBFT struct {
	sid	interface {
	}
	pid	interface {
	}
	B	interface {
	}
	N	interface {
	}
	f	interface {
	}
	sPK	interface {
	}
	sSK	interface {
	}
	ePK	interface {
	}
	eSK	interface {
	}
	_send	interface {
	}
	_recv	interface {
	}
	round			int
	transaction_buffer	[]interface {
	}
	_per_round_recv	map[int]interface {
	}
	_recv_thread	interface {
	}
}

func NewHoneyBadgerBFT(sid interface {
}, pid interface {
}, B interface {
}, N interface {
}, f interface {
}, sPK interface {
}, sSK interface {
}, ePK interface {
}, eSK interface {
}, send interface {
}, recv interface {
}) (self *HoneyBadgerBFT) {
	self = new(HoneyBadgerBFT)
	self.sid = sid
	self.pid = pid
	self.B = B
	self.N = N
	self.f = f
	self.sPK = sPK
	self.sSK = sSK
	self.ePK = ePK
	self.eSK = eSK
	self._send = send
	self._recv = recv
	self.round = 0
	self.transaction_buffer = []interface {
	}{}
	self._per_round_recv = map[interface {
	}]interface {
	}{}
	return
}
func (self *HoneyBadgerBFT) submit_tx(tx interface {
}) {
	"Appends the given transaction to the transaction buffer.\n\n        :param tx: Transaction to append to the buffer.\n        "
	fmt.Println("submit_tx", self.pid, tx)
	self.transaction_buffer = append(self.transaction_buffer, tx)
}
func (self *HoneyBadgerBFT) run() {
	"Run the HoneyBadgerBFT protocol."
	_recv := func() {
		"Receive messages."
		for {
			sender, [2]int{r, msg} := self._recv()
			if !func() bool {
				_, ok := self._per_round_recv[r]
				return ok
			}() {
				if !(r >= self.round) {
					panic(errors.New("AssertionError"))
				}
				self._per_round_recv[r] = Queue()
			}
			_recv := self._per_round_recv[r]
			if &_recv != &nil {
				_recv.put([2]interface {
				}{sender, msg})
			}
		}
	}
	/** todo: fix bugs here. Gevent => goroutine
	self._recv_thread = go _recv
	for {
		r := self.round
		if !func() bool {
			_, ok := self._per_round_recv[r]
			return ok
		}() {
			self._per_round_recv[r] = Queue()
		}
		tx_to_send := self.transaction_buffer[:self.B]
		_make_send := func(r int) {
			_send := func(j interface {
			}, o interface {
			}) {
				self._send(j, [2]interface {
				}{r, o})
			}
			_send
		}
		send_r := _make_send(r)
		recv_r := self._per_round_recv[r].get
		new_tx := self._run_round(r, tx_to_send[0], send_r, recv_r)
		fmt.Println("new_tx:", new_tx)
		self.transaction_buffer = func() (elts []interface {
		}) {
			for _, _tx := range self.transaction_buffer {
				if func() int {
					for i, v := range new_tx {
						if v == _tx {
							return i
						}
					}
					return -1
				}() == -1 {
					elts = append(elts, _tx)
				}
			}
			return
		}()
		self.round += 1
		if self.round >= 3 {
			break
		}
	}
}
func (self *HoneyBadgerBFT) _run_round(r interface {
}, tx_to_send interface {
}, send interface {
}, recv interface {
}) {
	"Run one protocol round.\n\n        :param int r: round id\n        :param tx_to_send: Transaction(s) to process.\n        :param send:\n        :param recv:\n        "
	sid := string(self.sid) + ":" + fmt.Sprintf("%v", r)
	pid := self.pid
	N := self.N
	f := self.f
	broadcast := func(o [3]string) {
		"Multicast the given input ``o``.\n\n            :param o: Input to multicast.\n            "
		for j := 0; j < N; j++ {
			send(j, o)
		}
	}
	coin_recvs := func(repeated []interface {
	}, n int) (result []interface {
	}) {
		for i := 0; i < n; i++ {
			result = append(result, repeated...)
		}
		return result
	}([]interface {
	}{nil}, N)
	aba_recvs := func(repeated []interface {
	}, n int) (result []interface {
	}) {
		for i := 0; i < n; i++ {
			result = append(result, repeated...)
		}
		return result
	}([]interface {
	}{nil}, N)
	rbc_recvs := func(repeated []interface {
	}, n int) (result []interface {
	}) {
		for i := 0; i < n; i++ {
			result = append(result, repeated...)
		}
		return result
	}([]interface {
	}{nil}, N)
	aba_inputs := func() (elts []interface {
	}) {
		for _ := 0; _ < N; _++ {
			elts = append(elts, Queue(1))
		}
		return
	}()
	aba_outputs := func() (elts []interface {
	}) {
		for _ := 0; _ < N; _++ {
			elts = append(elts, Queue(1))
		}
		return
	}()
	rbc_outputs := func() (elts []interface {
	}) {
		for _ := 0; _ < N; _++ {
			elts = append(elts, Queue(1))
		}
		return
	}()
	my_rbc_input := Queue(1)
	fmt.Println(pid, r, "tx_to_send:", tx_to_send)
	_setup := func(j int) {
		"Setup the sub protocols RBC, BA and common coin.\n\n            :param int j: Node index for which the setup is being done.\n            "
		coin_bcast := func(o interface {
		}) {
			"Common coin multicast operation.\n\n                :param o: Value to multicast.\n                "
			broadcast([3]string{"ACS_COIN", j, o})
		}
		coin_recvs[j] = Queue()
		coin := shared_coin(sid+"COIN"+fmt.Sprintf("%v", j), pid, N, f, self.sPK, self.sSK, coin_bcast, coin_recvs[j].get)
		aba_bcast := func(o interface {
		}) {
			"Binary Byzantine Agreement multicast operation.\n\n                :param o: Value to multicast.\n                "
			broadcast([3]string{"ACS_ABA", j, o})
		}
		aba_recvs[j] = Queue()
		// TODO:
		//gevent.spawn(binaryagreement, sid+"ABA"+fmt.Sprintf("%v", j), pid, N, f, coin, aba_inputs[j].get, aba_outputs[j].put_nowait, aba_bcast, aba_recvs[j].get)
		rbc_send := func(k interface {
		}, o interface {
		}) {
			"Reliable broadcast operation.\n\n                :param o: Value to broadcast.\n                "
			send(k, [3]string{"ACS_RBC", j, o})
		}
		rbc_input := func() interface {
		} {
			if reflect.DeepEqual(j, pid) {
				return my_rbc_input.get
			}
			return nil
		}()
		rbc_recvs[j] = Queue()
		rbc := gevent.spawn(reliablebroadcast, sid+"RBC"+fmt.Sprintf("%v", j), pid, N, f, j, rbc_input, rbc_recvs[j].get, rbc_send)
		rbc_outputs[j] = rbc.get
	}
	for j := 0; j < N; j++ {
		_setup(j)
	}
	tpke_bcast := func(o interface {
	}) {
		"Threshold encryption broadcast."
		broadcast([3]interface {
		}{"TPKE", 0, o})
	}
	tpke_recv := Queue()
	acs := gevent.spawn(commonsubset, pid, N, f, rbc_outputs, func() (elts []interface {
	}) {
		for _, _ := range aba_inputs {
			elts = append(elts, _.put_nowait)
		}
		return
	}(), func() (elts []interface {
	}) {
		for _, _ := range aba_outputs {
			elts = append(elts, _.get)
		}
		return
	}())
	recv_queues := BroadcastReceiverQueues(coin_recvs, aba_recvs, rbc_recvs, tpke_recv)
	gevent.spawn(broadcast_receiver_loop, recv, recv_queues)
	_input := Queue(1)
	_input.put(tx_to_send)
	honeybadger_block(pid, self.N, self.f, self.ePK, self.eSK, _input.get, my_rbc_input.put_nowait, acs.get, tpke_bcast, tpke_recv.get)
}**/