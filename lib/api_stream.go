package sebak

import (
	"fmt"
	"net/http"
	"strings"

	common "boscoin.io/sebak/lib/common"
	observable "github.com/GianlucaGuarini/go-observable"
)

const DefaultContentType = "application/json"

type EventStream struct {
	contentType string
	renderFunc  RenderFunc
	request     *http.Request
	writer      http.ResponseWriter
	flusher     http.Flusher
	err         error
}

type RenderFunc func(args ...interface{}) ([]byte, error)

var RenderSerializableFunc = func(args ...interface{}) ([]byte, error) {
	s, ok := args[1].(common.Serializable)
	if !ok {
		return nil, fmt.Errorf("this is not serializable") // TODO(anarcher): Error type
	}

	bs, err := s.Serialize()
	if err != nil {
		return nil, err
	}
	return bs, nil
}

func NewDefaultEventStream(w http.ResponseWriter, r *http.Request) *EventStream {
	return NewEventStream(w, r, RenderSerializableFunc, DefaultContentType)
}

func NewEventStream(w http.ResponseWriter, r *http.Request, renderFunc RenderFunc, ct string) *EventStream {
	es := &EventStream{
		request:     r,
		writer:      w,
		renderFunc:  renderFunc,
		contentType: ct,
	}

	w.Header().Set("Content-Type", es.contentType)

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		es.err = fmt.Errorf("http: can't do chunked response ")
	} else {
		es.flusher = flusher
	}

	return es
}

func (s *EventStream) Render(args ...interface{}) {
	if s.err != nil {
		return
	}
	var bs []byte
	var renderArgs []interface{}
	renderArgs = append(renderArgs, "pre")
	renderArgs = append(renderArgs, args...)
	if payload, err := s.renderFunc(renderArgs...); err != nil {
		bs = s.errMessage(err)
	} else {
		bs = payload
	}

	fmt.Fprintf(s.writer, "%s\n", bs)
	s.flusher.Flush()
}

func (s *EventStream) Run(ob *observable.Observable, events ...string) {
	s.Start(ob, events...)()
}

func (s *EventStream) Start(ob *observable.Observable, events ...string) func() {
	if s.err != nil {
		http.Error(s.writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return func() {}
	}

	event := strings.Join(events, " ")
	msg := make(chan []byte)
	stop := make(chan struct{})

	onFunc := func(args ...interface{}) {
		var (
			payload []byte
			err     error
		)

		if len(args) > 1 {
			payload, err = s.renderFunc(args...)
		} else {
			var as []interface{}
			as = append(as, event)
			as = append(as, args...)
			payload, err = s.renderFunc(as...)
		}

		if err != nil {
			//TODO(anarcher): We need error representation (e.g. HTTPError type?)
			msg <- s.errMessage(err)
		}
		select {
		case msg <- payload:
		case <-stop:
			return
		}
	}
	ob.On(event, onFunc)

	return func() {
		defer ob.Off(event, onFunc)

		for {
			select {
			case payload := <-msg:
				fmt.Fprintf(s.writer, "%s\n", payload)
				s.flusher.Flush()
			case <-s.request.Context().Done():
				close(stop)
				return
			}
		}
	}
}

func (s *EventStream) errMessage(err error) []byte {
	return []byte(fmt.Sprintf("{ \"err\": \"%s\"}", err.Error()))
}
