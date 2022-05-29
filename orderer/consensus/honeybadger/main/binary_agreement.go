package main

import (
	"fmt"
	tcrsa "github.com/niclabs/tcrsa"
	"strconv"
	"strings"
)

// place-holder function that is cross-process communication.
// func broadcast(prefix string, round int, value int, sender int) {
// 	fmt.Println("Broadcasting " + prefix + ", round: " + strconv.Itoa(round) + ", value: " + strconv.Itoa(value) + ", sender: " + strconv.Itoa(sender))
// }

// place-holder function that is get coin.
func get_coin() int {
	return 1
}

type message struct {
	prefix   string
	round    int
	value    int
	sender   int
	receiver int
}

// The reason we need message_string is because it is easier to communicate with other processes through strings.
// parse message_string to message object.
func parse_message(message_string string) message {
	split := strings.Split(message_string, "_")
	round, _ := strconv.ParseInt(split[1], 10, 0)
	value, _ := strconv.ParseInt(split[2], 10, 0)
	sender, _ := strconv.ParseInt(split[3], 10, 0)
	receiver, _ := strconv.ParseInt(split[4], 10, 0)
	m := message{
		prefix:   split[0],
		round:    int(round),
		value:    int(value),
		sender:   int(sender),
		receiver: int(receiver),
	}
	return m
}

// parse message object to message_string
func construct_aba_message(pid int, m message) hbMessage {
	msg := m.prefix + "_" + strconv.Itoa(m.round) + "_" + strconv.Itoa(m.value) + "_" + strconv.Itoa(m.sender) + "_" + strconv.Itoa(m.receiver)
	msgType := "ABA"
	sender := pid
	return hbMessage{
		msgType: msgType,
		sender:  sender,
		msg:     msg,
	}
}

func construct_message(prefix string, round int, value int, sender int, receiver int) message {
	m := message{
		prefix:   prefix,
		round:    round,
		value:    value,
		sender:   sender,
		receiver: receiver,
	}
	return m
}

func get_count(msgs_received []message, r int, value int, prefix string) int {
	seen_sender := make(map[int]bool)
	count := 0
	for _, msg := range msgs_received {
		if msg.round != r || msg.value != value || seen_sender[msg.sender] || msg.prefix != prefix {
			continue
		}
		count++
		seen_sender[msg.sender] = true
	}
	return count
}

func has_received_aux_already(msg message, msgs_received []message) bool {
	has_received := false
	if msg.prefix != "aux" {
		return false
	}
	for _, msg_received := range msgs_received {
		if msg.round == msg_received.round && msg.prefix == msg_received.prefix && msg.sender == msg_received.sender {
			has_received = true
			break
		}
	}
	return has_received
}

/**
pid : process id
N : total nodes
f : faulty nodes
coin : func(int) int Takes in round number and outputs coin result.
input: Channel containing input to binaryagreement
decide: Channel to put decision of binaryagreement on input
broadcast: func(string) Takes in a string to broadcast
receive: Channel to receive binaryagreement string messages from other nodes.
*/
func binaryagreement(
	sid string,
	pid int,
	N int,
	f int,
	leader int,
	input chan int,
	decide chan int,
	// broadcast func(string),
	receive chan string,
	coin_sid string,
	coin_pid int,
	coin_N int,
	coin_f int,
	coin_meta tcrsa.KeyMeta,
	coin_keyShare tcrsa.KeyShare,
	// coin_broadcast func(string),
	coin_receive chan string,
) {
	r := 0
	est := <-input
	msgs_received := make([]message, 0)
	for {
		fmt.Println("In round " + strconv.Itoa(r))

		broadcast(construct_aba_message(leader, construct_message("bval", r, est, pid, -1)))
		// Count your own message/vote too.
		msgs_received = append(msgs_received, message{prefix: "bval", round: r, value: est, sender: pid, receiver: pid})

		bin_values := make([]int, 0)

		b_value := -1 // will be the result I vote for in this round
		for {
			msg := <-receive
			fmt.Println("Parsing message " + msg)
			msg_object := parse_message(msg)

			// Check that evil actors don't send two conflicting aux messages. If so ignore the message.
			if msg_object.prefix == "aux" {
				if has_received_aux_already(msg_object, msgs_received) {
					fmt.Println("Already received aux message in round " + strconv.Itoa(msg_object.round) + " from sender " + strconv.Itoa(msg_object.sender))
					continue
				}
			}

			// Check that the message received is for this round.
			if msg_object.round != r {
				fmt.Println("Ignore message because not for this round.")
				continue
			}

			msgs_received = append(msgs_received, msg_object)

			if msg_object.prefix == "bval" {
				count := get_count(msgs_received, r, msg_object.value, "bval")
				if count >= f+1 {
					fmt.Println("Has received more than f+1 bval messages with value " + strconv.Itoa(msg_object.value) + " at round " + strconv.Itoa(r))
					broadcast(construct_aba_message(leader, construct_message("bval", r, msg_object.value, pid, -1)))
					// Count your own message/vote too.
					msgs_received = append(msgs_received, message{prefix: "bval", round: r, value: msg_object.value, sender: pid, receiver: pid})
				}
				count = get_count(msgs_received, r, msg_object.value, "bval")
				if count >= 2*f+1 {
					fmt.Println("Has received more than 2f+1 bval messages with value " + strconv.Itoa(msg_object.value) + " at round " + strconv.Itoa(r))
					in_bin_values := false
					for _, v := range bin_values {
						if v == msg_object.value {
							in_bin_values = true
						}
					}
					if !in_bin_values {
						fmt.Println("Value " + strconv.Itoa(msg_object.value) + " not yet in bin_values. Adding now.")
						bin_values = append(bin_values, msg_object.value)
					}
				}
			}

			if len(bin_values) == 0 {
				continue
			}
			if b_value == -1 {
				b_value = bin_values[0]
				fmt.Println("b set to be " + strconv.Itoa(b_value))
				broadcast(construct_aba_message(leader, construct_message("aux", r, b_value, pid, -1)))
				// Count your own message/vote too.
				msgs_received = append(msgs_received, message{prefix: "aux", round: r, value: b_value, sender: pid, receiver: pid})
			}

			count_0 := get_count(msgs_received, r, 0, "aux")
			count_1 := get_count(msgs_received, r, 1, "aux")

			has_0 := false
			has_1 := false
			for _, x := range bin_values {
				if x == 0 {
					has_0 = true
				}
				if x == 1 {
					has_1 = true
				}
			}
			has_0_int := 0
			has_1_int := 0
			if has_0 {
				has_0_int = 1
			}

			if has_1 {
				has_1_int = 1
			}
			total_acceptable_messages := has_0_int*count_0 + has_1_int*count_1
			if total_acceptable_messages < N-f {
				fmt.Println("Less than N-f messages, whose value is a subset of bin_values.")
				continue
			}
			fmt.Println("Received at least N-f messsages, whose value is a subset of bin_values.")
			fmt.Println("Calling get coin.")
			coin_val := shared_coin(
				coin_sid,
				coin_pid,
				coin_N,
				coin_f,
				leader,
				coin_meta,
				coin_keyShare,
				// coin_broadcast,
				coin_receive,
				r)
			if (b_value == 0 && has_0_int*count_0 >= N-f) || (b_value == 1 && has_1_int*count_1 >= N-f) {
				fmt.Println("vals = {b}")
				est = b_value
				if b_value == coin_val {
					decide <- b_value
				}
			} else {
				fmt.Println("vals != {b}")
				est = coin_val
			}
			break
		}
		r++
	}
}

// id is the identification of the current process.
// input is the value that we propose (0 or 1).
// messages is the channel that binary agreement will be using to RECEIVE other messages.
// f is faulty nodes
// N is total nodes
// func binary_agreement(id int, input int, messages chan string, f int, N int) int {
// 	r := 0
// 	est := input
// 	msgs_received := make([]message, 0)
// 	for {
// 		fmt.Println("In round " + strconv.Itoa(r))

// 		broadcast("bval_" + strconv.Itoa(r) + "_" + strconv.Itoa(est) + "_" + strconv.Itoa(id))
// 		// Count your own message/vote too.
// 		msgs_received = append(msgs_received, message{prefix: "bval", round: r, value: est, sender: id, receiver: id})

// 		bin_values := make([]int, 0)

// 		b_value := -1 // will be the result I vote for in this round
// 		for {
// 			msg := <-messages
// 			fmt.Println("Parsing message " + msg)
// 			msg_object := parse_message(msg)

// 			// Check that evil actors don't send two conflicting aux messages. If so ignore the message.
// 			if msg_object.prefix == "aux" {
// 				if has_received_aux_already(msg_object, msgs_received) {
// 					fmt.Println("Already received aux message in round " + strconv.Itoa(msg_object.round) + " from sender " + strconv.Itoa(msg_object.sender))
// 					continue
// 				}
// 			}

// 			// Check that the message received is for this round.
// 			if msg_object.round != r {
// 				fmt.Println("Ignore message because not for this round.")
// 				continue
// 			}

// 			msgs_received = append(msgs_received, msg_object)

// 			if msg_object.prefix == "bval" {
// 				count := get_count(msgs_received, r, msg_object.value, "bval")
// 				if count >= f+1 {
// 					fmt.Println("Has received more than f+1 bval messages with value " + strconv.Itoa(msg_object.value) + " at round " + strconv.Itoa(r))
// 					broadcast("bval", r, msg_object.value, id)
// 					// Count your own message/vote too.
// 					msgs_received = append(msgs_received, message{prefix: "bval", round: r, value: msg_object.value, sender: id, receiver: id})
// 				}
// 				count = get_count(msgs_received, r, msg_object.value, "bval")
// 				if count >= 2*f+1 {
// 					fmt.Println("Has received more than 2f+1 bval messages with value " + strconv.Itoa(msg_object.value) + " at round " + strconv.Itoa(r))
// 					in_bin_values := false
// 					for _, v := range bin_values {
// 						if v == msg_object.value {
// 							in_bin_values = true
// 						}
// 					}
// 					if !in_bin_values {
// 						fmt.Println("Value " + strconv.Itoa(msg_object.value) + " not yet in bin_values. Adding now.")
// 						bin_values = append(bin_values, msg_object.value)
// 					}
// 				}
// 			}

// 			if len(bin_values) == 0 {
// 				continue
// 			}
// 			if b_value == -1 {
// 				b_value = bin_values[0]
// 				fmt.Println("b set to be " + strconv.Itoa(b_value))
// 				broadcast("aux", r, b_value, id)
// 				// Count your own message/vote too.
// 				msgs_received = append(msgs_received, message{prefix: "aux", round: r, value: b_value, sender: id, receiver: id})
// 			}

// 			count_0 := get_count(msgs_received, r, 0, "aux")
// 			count_1 := get_count(msgs_received, r, 1, "aux")

// 			has_0 := false
// 			has_1 := false
// 			for _, x := range bin_values {
// 				if x == 0 {
// 					has_0 = true
// 				}
// 				if x == 1 {
// 					has_1 = true
// 				}
// 			}
// 			has_0_int := 0
// 			has_1_int := 0
// 			if has_0 {
// 				has_0_int = 1
// 			}

// 			if has_1 {
// 				has_1_int = 1
// 			}
// 			total_acceptable_messages := has_0_int*count_0 + has_1_int*count_1
// 			if total_acceptable_messages < N-f {
// 				fmt.Println("Less than N-f messages, whose value is a subset of bin_values.")
// 				continue
// 			}
// 			fmt.Println("Received at least N-f messsages, whose value is a subset of bin_values.")
// 			fmt.Println("Calling get coin.")
// 			coin_val := get_coin()
// 			if (b_value == 0 && has_0_int*count_0 >= N-f) || (b_value == 1 && has_1_int*count_1 >= N-f) {
// 				fmt.Println("vals = {b}")
// 				est = b_value
// 				if b_value == coin_val {
// 					return b_value
// 				}
// 			} else {
// 				fmt.Println("vals != {b}")
// 				est = coin_val
// 			}
// 			break
// 		}
// 		r++
// 	}
// }

// func main() {
// 	messages := make(chan string)

// 	// Test case 1, 4 nodes, 1 faulty. Assume faulty node is just silent
// 	fmt.Println("Test 1")
// 	go func() {
// 		messages <- "bval_0_1_1_0" // receive message of value 1 from node 1
// 		messages <- "bval_0_1_2_0" // receive message of value 1 from node 2
// 		messages <- "aux_0_1_1_0"  // receive message of value 1 from node 1
// 		messages <- "aux_0_1_2_0"  // receive message of value 1 from node 2
// 	}()
// 	fmt.Println(binary_agreement(0, 0, messages, 1, 4))

// 	// Test case 2, 4 nodes, 1 faulty. Assume faulty node is advocating fault results.
// 	fmt.Println("\n\n\n\n\n\n\n\n\n\n\nTest 2")
// 	go func() {
// 		// Bad actor sending conflicting messages.
// 		messages <- "bval_0_0_3_0" // receive message of value 0 from node 3
// 		messages <- "bval_0_0_3_0" // receive message of value 0 from node 3
// 		messages <- "bval_0_0_3_0" // receive message of value 0 from node 3

// 		messages <- "bval_0_1_1_0" // receive message of value 1 from node 1
// 		messages <- "bval_0_1_2_0" // receive message of value 1 from node 2

// 		// Faulty messages
// 		messages <- "bval_0_0_3_0" // receive message of value 0 from node 3
// 		messages <- "bval_0_0_3_0" // receive message of value 0 from node 3
// 		messages <- "bval_0_0_3_0" // receive message of value 0 from node 3
// 		messages <- "bval_0_0_3_0" // receive message of value 0 from node 3
// 		messages <- "bval_0_0_3_0" // receive message of value 0 from node 3
// 		messages <- "bval_0_0_3_0" // receive message of value 0 from node 3
// 		messages <- "bval_0_0_3_0" // receive message of value 0 from node 3
// 		messages <- "aux_0_0_3_0"  // receive message of value 0 from node 3
// 		messages <- "aux_0_0_3_0"  // receive message of value 0 from node 3
// 		messages <- "aux_0_0_3_0"  // receive message of value 0 from node 3
// 		messages <- "aux_0_0_3_0"  // receive message of value 0 from node 3
// 		messages <- "aux_0_0_3_0"  // receive message of value 0 from node 3
// 		messages <- "aux_0_0_3_0"  // receive message of value 0 from node 3

// 		messages <- "aux_0_1_1_0" // receive message of value 1 from node 1

// 		// Bad actor sending conflicting messages.
// 		messages <- "bval_0_0_3_0" // receive message of value 0 from node 3
// 		messages <- "bval_0_0_3_0" // receive message of value 0 from node 3
// 		messages <- "bval_0_0_3_0" // receive message of value 0 from node 3
// 		messages <- "bval_0_0_3_0" // receive message of value 0 from node 3
// 		messages <- "bval_0_0_3_0" // receive message of value 0 from node 3
// 		messages <- "bval_0_0_3_0" // receive message of value 0 from node 3
// 		messages <- "bval_0_0_3_0" // receive message of value 0 from node 3
// 		messages <- "aux_0_0_3_0"  // receive message of value 0 from node 3
// 		messages <- "aux_0_0_3_0"  // receive message of value 0 from node 3
// 		messages <- "aux_0_0_3_0"  // receive message of value 0 from node 3
// 		messages <- "aux_0_0_3_0"  // receive message of value 0 from node 3
// 		messages <- "aux_0_0_3_0"  // receive message of value 0 from node 3
// 		messages <- "aux_0_0_3_0"  // receive message of value 0 from node 3

// 		messages <- "aux_0_1_2_0" // receive message of value 1 from node 2

// 	}()
// 	fmt.Println(binary_agreement(0, 0, messages, 1, 4))

// 	fmt.Println("\n\n\n\n\n\n\n\n\n\n\nTest 3")
// 	// Conflicting aux messages that need to go to round two and use common coin to decide.
// 	go func() {
// 		// First you broadcast 0.
// 		// But node 1, 2 believes in 1.
// 		// Node 3 also believes in 0.
// 		messages <- "bval_0_1_1_0" // receive message of value 1 from node 1
// 		messages <- "bval_0_1_2_0" // receive message of value 1 from node 2
// 		messages <- "bval_0_0_3_0" // receive message of value 1 from node 3

// 		messages <- "bval_0_0_1_0" // receive message of value 0 from node 1
// 		messages <- "bval_0_0_2_0" // receive message of value 0 from node 2
// 		messages <- "bval_0_1_3_0" // receive message of value 0 from node 3

// 		messages <- "aux_0_0_1_0" // receive message of value 0 from node 1
// 		messages <- "aux_0_0_2_0" // receive message of value 0 from node 2
// 		messages <- "aux_0_0_3_0" // receive message of value 0 from node 3

// 		// Others believe in 0 and I believe in 1. So there is a conflict. But common coin says 1 so continue with 1 in next round.

// 		messages <- "bval_1_1_1_0" // receive message of value 1 from node 1
// 		messages <- "bval_1_1_2_0" // receive message of value 1 from node 2
// 		messages <- "bval_1_1_3_0" // receive message of value 1 from node 3

// 		messages <- "aux_1_1_1_0" // receive message of value 0 from node 1
// 		messages <- "aux_1_1_2_0" // receive message of value 0 from node 2
// 		messages <- "aux_1_1_3_0" // receive message of value 0 from node 3
// 	}()
// 	fmt.Println(binary_agreement(0, 0, messages, 1, 4))
// }
