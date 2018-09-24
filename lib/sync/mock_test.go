package sync

type MockConsumer struct {
	msg  chan *Message
	resp chan *Response
	done chan struct{}
}

func NewMockConsumer() *MockConsumer {
	c := &MockConsumer{
		msg:  make(chan *Message),
		resp: make(chan *Response),
		done: make(chan struct{}),
	}
	return c
}

func (c *MockConsumer) Stop() error {
	close(c.done)
	return nil
}

func (c *MockConsumer) Consume(msgc <-chan *Message) error {
	go func() {
		for {
			select {
			case m := <-msgc:
				c.msg <- m
			case <-c.done:
				return
			}
		}
	}()

	return nil
}

func (c *MockConsumer) Response() <-chan *Response {
	return c.resp
}

func (c *MockConsumer) Message() <-chan *Message {
	return c.msg
}
