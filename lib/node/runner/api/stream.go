package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/GianlucaGuarini/go-observable"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common/observer"
	"boscoin.io/sebak/lib/errors"
	"boscoin.io/sebak/lib/network/httputils"
	"boscoin.io/sebak/lib/node/runner/api/resource"
)

// DefaultContentType is "application/json"
const DefaultContentType = "application/json"

func (api NetworkHandlerAPI) PostSubscribeHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if !httputils.IsEventStream(r) {
		httputils.WriteJSONError(w, errors.BadRequestParameter)
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		httputils.WriteJSONError(w, errors.BadRequestParameter)
		return
	}
	var requestParams []observer.Conditions
	if err := json.Unmarshal(body, &requestParams); err != nil {
		httputils.WriteJSONError(w, errors.BadRequestParameter)
		return
	}

	var events []string
	for _, conditions := range requestParams {
		events = append(events, conditions.Event())
	}

	renderFunc := func(args ...interface{}) ([]byte, error) {
		if len(args) <= 1 {
			return nil, fmt.Errorf("render: value is empty") //TODO(anarcher): Error type
		}
		i := args[1]

		if i == nil {
			return []byte{}, nil
		}

		switch v := i.(type) {
		case *block.BlockAccount:
			r := resource.NewAccount(v)
			return json.Marshal(r.Resource())
		case *block.BlockTransaction:
			tp, err := block.GetTransactionPool(api.storage, v.Hash)
			if err != nil {
				return nil, err
			}
			r := resource.NewTransaction(v, tp.Transaction())
			return json.Marshal(r.Resource())
		}

		return json.Marshal(i)
	}

	es := NewEventStream(w, r, renderFunc, DefaultContentType)
	es.Render(nil)
	es.Run(observer.ResourceObserver, events...)
}

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
	stop        chan struct{}
}

type RenderFunc func(args ...interface{}) ([]byte, error)

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

// NewDefaultEventStream returns *EventStream with RenderJSONFunc and DefaultContentType
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
	s.stop = make(chan struct{})

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
		case <-s.stop:
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
				close(s.stop)
				return
			}
		}
	}
}
func (s *EventStream) Stop() {
	close(s.stop)
}

func (s *EventStream) errMessage(err error) []byte {

	p := httputils.NewErrorProblem(err, httputils.StatusCode(err))
	b, err := json.Marshal(p)
	if err != nil {
		b = []byte{}
	}
	return b
}
