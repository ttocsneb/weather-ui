package util

import "fmt"

/*
Channel Multiplexer

Allows for a generator to produce messages that will be sent to subscribers.
*/
type ChanMultiplex[T any] struct {
	chans   []chan T
	done    chan struct{}
	routine func(*ChanMultiplex[T], chan struct{})
}

/*
Create a new multiplexer.

The function provided is executed as its own goroutine. Messages may be sent to
subscribers by calling Notify. When the routine is finished, then a message
will be sent to the provided channel

	NewChanMultiplex(func(m *ChanMultiplex[int], done chan struct{}) {
		for {
			m.Notify(5)
			select {
			case <-done:
				return
			default:
			}
		}

	})
*/
func NewChanMultiplex[T any](goroutine func(*ChanMultiplex[T], chan struct{})) *ChanMultiplex[T] {
	return &ChanMultiplex[T]{
		chans:   []chan T{},
		done:    nil,
		routine: goroutine,
	}
}

/*
Notify all subscribers
*/
func (self *ChanMultiplex[T]) Notify(val T) {
	fmt.Printf("Notifying subscribers... %v\n", val)
	for _, ch := range self.chans {
		ch <- val
	}
}

/*
Subscribe to the multiplexer

The Unsubscribe function should be called when finished with the channel
*/
func (self *ChanMultiplex[T]) Subscribe() chan T {
	if self.done == nil {
		self.done = make(chan struct{})
	}

	ch := make(chan T)

	self.chans = append(self.chans, ch)

	if len(self.chans) == 1 {
		go self.routine(self, self.done)
	}

	return ch
}

/*
Unsubscribe a channel from the multiplexer
*/
func (self *ChanMultiplex[T]) Unsubscribe(ch chan T) {
	for i, c := range self.chans {
		if c == ch {
			fmt.Println("Found channel to unsub")
			fmt.Printf("before: %v\n", self.chans)
			self.chans = append(self.chans[:i], self.chans[i+1:]...)
			fmt.Printf("after: %v\n", self.chans)
			close(ch)
			fmt.Println("Unsubscribing...")
			if len(self.chans) == 0 {
				fmt.Println("Closing...")
				self.done <- struct{}{}
				self.done = nil
			}
			return
		}
	}
	fmt.Println("Unable to unsubscribe")
}

/*
Close the multiplexer
*/
func (self *ChanMultiplex[T]) Close() {
	fmt.Println("Closing the multiplexer")
	self.done <- struct{}{}
	for _, c := range self.chans {
		close(c)
	}
	self.chans = []chan T{}
	self.done = nil
}
