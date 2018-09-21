package sync

var _ ValidationLayer = (*MultiConsumer)(nil)

type MultiConsumer struct {
	consumers []Consumer

	message  <-chan *Message
	response chan *Response

	stops []chan chan struct{}
}

func NewMultiConsumer(cs ...Consumer) *MultiConsumer {
	c := &MultiConsumer{
		consumers: cs,
		message:   nil,
		response:  make(chan *Response),
		stops:     make([]chan chan struct{}, 0, len(cs)),
	}

	return c
}

func (c *MultiConsumer) Stop() error {
	for _, stop := range c.stops {
		c := make(chan struct{})
		stop <- c
		<-c
	}

	return nil
}

func (c *MultiConsumer) Consume(msg <-chan *Message) error {
	c.message = msg
	return nil
}

func (c *MultiConsumer) Response() <-chan *Response {
	return c.response
}

func (c *MultiConsumer) startConsumers() {
	for _, cs := range c.consumers {
		stop := make(chan chan struct{})
		c.startConsumer(cs, stop)
		c.stops = append(c.stops, stop)
	}
}

func (c *MultiConsumer) startConsumer(cs Consumer, stop chan chan struct{}) {
	cs.Consume(c.message)
	respCh := cs.Response()
	go func() {
		for {
			select {
			case resp := <-respCh:
				c.response <- resp
			case s := <-stop:
				close(s)
				return
			}
		}
	}()
}
