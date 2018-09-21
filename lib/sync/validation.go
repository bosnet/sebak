package sync

var _ ValidationLayer = (*Validation)(nil)

type Validation struct {
	consumeMessage <-chan *Message
	response       chan *Response

	numWorkers int

	stop chan chan struct{}
}

func NewValidation(numWorkers int) *Validation {
	v := &Validation{
		consumeMessage: nil,
		response:       make(chan *Response),
		numWorkers:     numWorkers,
		stop:           make(chan chan struct{}),
	}

	return v
}

func (v *Validation) Stop() error {
	c := make(chan struct{})
	v.stop <- c
	<-c
	return nil
}

func (v *Validation) Consume(msg <-chan *Message) error {
	v.consumeMessage = msg
	return nil
}

func (v *Validation) Response() <-chan *Response {
	return v.response
}
