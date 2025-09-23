package framework

type Semaphore struct {
	ch chan struct{}
}

func NewSemaphore() *Semaphore {
	return &Semaphore{
		ch: make(chan struct{}),
	}
}

func (s *Semaphore) Release() {
	close(s.ch)
}

func (s *Semaphore) Wait() {
	<-s.ch
}
