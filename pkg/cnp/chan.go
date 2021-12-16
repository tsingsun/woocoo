package cnp

import "sync"

// DoneChan a chan wait for done,like application or connection exit
type DoneChan struct {
	done chan struct{}
	once sync.Once
}

// NewDoneChan create a DoneChan
func NewDoneChan() *DoneChan {
	return &DoneChan{
		done: make(chan struct{}),
	}
}

// Close closes inner chan.
func (dc *DoneChan) Close() {
	dc.once.Do(func() {
		close(dc.done)
	})
}

// Done return inner chan
func (dc *DoneChan) Done() chan struct{} {
	return dc.done
}
