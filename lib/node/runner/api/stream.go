package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/network/httputils"
	observable "github.com/GianlucaGuarini/go-observable"
)

// DefaultContentType is "application/json"
const DefaultContentType = "application/json"

// EventStream handles chunked responses of a observable trigger
//
// renderFunc uses on observable.On() and Render function
type EventStream struct {
	contentType string
	renderFunc  RenderFunc
	request     *http.Request
	writer      http.ResponseWriter
	flusher     http.Flusher
	err         error
	rendered    bool
}

type RenderFunc func(args ...interface{}) ([]byte, error)

// RenderSerializableFunc takes common.Serializable and serialize it for rendering.
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

// NewDefaultEventStream uses RenderJSONFunc by default
var RenderJSONFunc = func(args ...interface{}) ([]byte, error) {
	if len(args) <= 1 {
		return nil, fmt.Errorf("render: value is empty") //TODO(anarcher): Error type
	}
	v := args[1]
	if v == nil {
		return nil, nil
	}
	bs, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return bs, nil
}

// NewDefaultEventStream returns *EventStream with RenderSerializableFunc and DefaultContentType
func NewDefaultEventStream(w http.ResponseWriter, r *http.Request) *EventStream {
	return NewEventStream(w, r, RenderJSONFunc, DefaultContentType)
}

// NewEventStream makes *EventStream and checks http.Flusher by type assertion.
func NewEventStream(w http.ResponseWriter, r *http.Request, renderFunc RenderFunc, ct string) *EventStream {
	es := &EventStream{
		request:     r,
		writer:      w,
		renderFunc:  renderFunc,
		contentType: ct,
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		es.err = fmt.Errorf("http: can't do chunked response ")
	} else {
		es.flusher = flusher
	}

	return es
}

// Render make a chunked response by using RenderFunc and flush it.
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

	if !s.rendered {
		s.writer.Header().Set("Content-Type", s.contentType)
		s.rendered = true
	}

	fmt.Fprintf(s.writer, "%s\n", bs)
	s.flusher.Flush()
}

// Run start observing events.
//
// Simple use case:
//
// 	event := fmt.Sprintf("address-%s", address)
// 	es := NewDefaultEventStream(w, r)
// 	es.Render(blk)
// 	es.Run(observer.BlockAccountObserver, event)
func (s *EventStream) Run(ob *observable.Observable, events ...string) {
	s.Start(ob, events...)()
}

// Start prepares for observing events and returns run func.
//
// In most case, Use Run instead of Start
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

	p := httputils.NewErrorProblem(err, httputils.StatusCode(err))
	b, err := json.Marshal(p)
	if err != nil {
		b = []byte{}
	}
	return b
}
