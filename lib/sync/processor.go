package sync

var _ FetchLayer = (*MultiProcessor)(nil)

type MultiProcessor struct {
	processors []Processor

	message  chan *Message
	response <-chan *Response

	incomingMessage <-chan *Message
	procsResponse   chan *Response

	stopProcs []chan chan struct{}
	stop      chan chan struct{}
}

func NewMultiProcessor(ps ...Processor) *MultiProcessor {
	p := &MultiProcessor{
		processors:      ps,
		message:         make(chan *Message),
		response:        nil,
		incomingMessage: nil,
		procsResponse:   make(chan *Response),
		stopProcs:       make([]chan chan struct{}, 0, len(ps)),
	}
	p.startProcessors()
	return p
}

// Stopper
func (p *MultiProcessor) Stop() error {
	for _, stop := range p.stopProcs {
		c := make(chan struct{})
		stop <- c
		<-c
	}
	return nil
}

// Producer
func (p *MultiProcessor) Produce(response <-chan *Response) {
	p.response = response
}

func (p *MultiProcessor) Message() <-chan *Message {
	return p.message
}

// Consumer
func (p *MultiProcessor) Consume(msg <-chan *Message) error {
	p.incomingMessage = msg
	return nil
}
func (p *MultiProcessor) Response() <-chan *Response {
	return p.procsResponse
}

func (p *MultiProcessor) startProcessors() {
	for _, proc := range p.processors {
		stop := make(chan chan struct{})
		p.startProcessor(proc, stop)
		p.stopProcs = append(p.stopProcs, stop)
	}
}

func (p *MultiProcessor) startProcessor(proc Processor, stop chan chan struct{}) {
	proc.Consume(p.incomingMessage)
	proc.Produce(p.response)
	msgCh := proc.Message()
	respCh := proc.Response()
	stopCh := stop
	go func() {
		select {
		case msg := <-msgCh:
			p.message <- msg
		case resp := <-respCh:
			p.procsResponse <- resp
		case c := <-stopCh:
			close(c)
			return
		}
	}()
}
