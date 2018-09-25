package sync

var (
	_ Consumer  = (*MockConsumer)(nil)
	_ Producer  = (*MockProducer)(nil)
	_ Processor = (*MockProcessor)(nil)
)

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

type MockProducer struct {
	msgc  chan *Message
	respc <-chan *Response
}

func NewMockProducer() *MockProducer {
	p := &MockProducer{
		msgc: make(chan *Message),
	}
	return p
}

func (p *MockProducer) Produce() <-chan *Message {
	return p.msgc
}

func (p *MockProducer) SetResponse(resp <-chan *Response) error {
	p.respc = resp
	return nil
}

func (p *MockProducer) GetResponse() <-chan *Response {
	return p.respc
}

type MockProcessor struct {
	p    *MockProducer
	c    *MockConsumer
	done chan struct{}
}

func (p *MockProcessor) Stop() error {
	p.c.Stop()
	close(p.done)

	return nil
}

func (p *MockProcessor) Produce() <-chan *Message {
	return p.p.Produce()
}
func (p *MockProcessor) SetResponse(resp <-chan *Response) error {
	return p.p.SetResponse(resp)
}

func (p *MockProcessor) Consume(msg <-chan *Message) error {
	return p.c.Consume(msg)

}
func (p *MockProcessor) Response() <-chan *Response {
	return p.c.Response()
}

func NewMockProcessor() *MockProcessor {
	p := NewMockProducer()
	c := NewMockConsumer()

	proc := &MockProcessor{
		p:    p,
		c:    c,
		done: make(chan struct{}),
	}

	go func() {
		for {
			select {
			case msg := <-c.msg:
				p.msgc <- msg
			case <-proc.done:
				return
			}
		}
	}()

	return proc
}
