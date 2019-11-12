package core

import (
	"sync"
)

// watchForInterrupts broadcasts a done signal to several output channels
func watchForInterrupts(doneChan <-chan struct{}, wg *sync.WaitGroup, outputChans ...chan<- struct{}) {
	defer wg.Done()

	if _, interrupt := <-doneChan; interrupt {
		for _, outputChan := range outputChans {
			outputChan <- struct{}{}
		}
	}
}
