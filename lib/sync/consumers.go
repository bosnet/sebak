package sync

var _ Validator = (*Consumers)(nil)

type Consumers struct {
	consumers []Consumer

	message  <-chan *Message
	response chan *Response

	stops []chan chan struct{}
}

func NewConsumers(cs ...Consumer) *Consumers {
	c := &Consumers{
		consumers: cs,
		message:   nil,
		response:  make(chan *Response),
		stops:     make([]chan chan struct{}, 0, len(cs)),
	}

	return c
}

func (c *Consumers) Stop() error {
	for _, stop := range c.stops {
		c := make(chan struct{})
		stop <- c
		<-c
	}

	return nil
}

func (c *Consumers) Consume(msg <-chan *Message) error {
	c.message = msg
	c.startConsumers()
	return nil
}

func (c *Consumers) Response() <-chan *Response {
	return c.response
}

func (c *Consumers) startConsumers() {
	for _, cs := range c.consumers {
		stop := make(chan chan struct{})
		go c.startConsumer(cs, stop)
		c.stops = append(c.stops, stop)
	}
}

func (c *Consumers) startConsumer(cs Consumer, stop chan chan struct{}) {
	cs.Consume(c.message)
	for {
		select {
		case resp := <-cs.Response():
			c.response <- resp
		case s := <-stop:
			close(s)
			return
		}
	}
}
