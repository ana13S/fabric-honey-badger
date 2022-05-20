package main

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func commonsubset(pid int, N int, f int, rbc_out []func() string, aba_in []func(x int), aba_out []func() int, t *testing.T) []string {

	// setup collector slices
	aba_inputted := make([]bool, N)
	aba_values := make([]int, N)
	rbc_values := make([]string, N)

	// recv_rbc kill channels
	recv_rbc_killChans := make([]chan string, N)
	for i := 0; i < N; i++ {
		recv_rbc_killChans[i] = make(chan string)
	}

	// recv_rbc join channels
	recv_rbc_joinChans := make([]chan string, N)
	for i := 0; i < N; i++ {
		recv_rbc_joinChans[i] = make(chan string)
	}

	recv_rbc := func(j int, killChan <-chan string, joinChan chan<- string) {

		for {
			select {

			// listen to killChan for kill signal
			case <-killChan:
				return

			// otherwise run
			default:
				// Receive output from reliable broadcast
				rbc_values[j] = rbc_out[j]()

				if !aba_inputted[j] {

					aba_inputted[j] = true

					aba_in[j](1)
				}

				// output "finished" to joinChan, can wait for joinChan as a blocker
				joinChan <- "finished"
			}
		}
	}

	// spawn recv_rbc goroutines
	for i := 0; i < N; i++ {
		go recv_rbc(i, recv_rbc_killChans[i], recv_rbc_joinChans[i])
	}

	recv_aba := func(j int, wg *sync.WaitGroup) {
		defer wg.Done()

		aba_values[j] = aba_out[j]()

		// count number of non-zero positions in aba_values
		arrSum := 0
		for i := 0; i < len(aba_values); i++ {
			arrSum = arrSum + aba_values[i]
		}

		if arrSum >= N-f {

			for i := 0; i < N; i++ {

				if !aba_inputted[i] {
					aba_inputted[i] = true

					aba_in[i](0)
				}
			}
		}
	}

	// equivalent of gevent spawn, joinall athreads
	var wg sync.WaitGroup

	for i := 0; i < N; i++ {
		wg.Add(1)
		recv_aba(i, &wg)
	}

	wg.Wait()

	// must have at least N-f committed
	// count number of non-zero positions in aba_values
	arrSum := 0
	for i := 0; i < len(aba_values); i++ {
		arrSum = arrSum + aba_values[i]
	}
	assert.GreaterOrEqual(t, arrSum, N-f)

	// wait for the corresponding broadcasts
	for j := 0; j < N; j++ {
		if aba_values[j] == 1 {

			// wait for jth recv_rbc goroutine to finish
			<-recv_rbc_joinChans[j]

			// nonetype error handling
			assert.NotEqual(t, rbc_values[j], "")
		} else {

			recv_rbc_killChans[j] <- "kill"
			rbc_values[j] = ""
		}
	}

	return rbc_values
}
