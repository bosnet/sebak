package sync

var _ Fetcher = (*Processors)(nil)

type Processors struct {
	processors []Processor

	message  chan *Message
	response <-chan *Response

	incomingMessage <-chan *Message
	procsResponse   chan *Response

	stopProcs []chan chan struct{}
	stop      chan chan struct{}
}

func NewProcessors(ps ...Processor) *Processors {
	p := &Processors{
		processors:      ps,
		message:         make(chan *Message),
		response:        nil,
		incomingMessage: nil,
		procsResponse:   make(chan *Response),
		stopProcs:       make([]chan chan struct{}, 0, len(ps)),
	}
	return p
}

// Stopper
func (p *Processors) Stop() error {
	for _, stop := range p.stopProcs {
		c := make(chan struct{})
		stop <- c
		<-c
	}
	return nil
}

// Producer
func (p *Processors) SetResponse(response <-chan *Response) error {
	p.response = response
	return nil
}

func (p *Processors) Produce() <-chan *Message {
	return p.message
}

// Consumer
func (p *Processors) Consume(msg <-chan *Message) error {
	p.incomingMessage = msg
	p.startProcessors()
	return nil
}
func (p *Processors) Response() <-chan *Response {
	return p.procsResponse
}

func (p *Processors) startProcessors() {
	for _, proc := range p.processors {
		stop := make(chan chan struct{})
		go p.startProcessor(proc, stop)
		p.stopProcs = append(p.stopProcs, stop)
	}
}

func (p *Processors) startProcessor(proc Processor, stop chan chan struct{}) {
	proc.Consume(p.incomingMessage)
	proc.SetResponse(p.response)
	for {
		select {
		case msg := <-proc.Produce():
			p.message <- msg
		case resp := <-proc.Response():
			p.procsResponse <- resp
		case c := <-stop:
			close(c)
			return
		}
	}
}
