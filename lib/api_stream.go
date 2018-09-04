package sebak

import (
	"fmt"
	"net/http"
	"strings"

	common "boscoin.io/sebak/lib/common"
	"github.com/GianlucaGuarini/go-observable"
)

type EventStream struct {
	Events      []string
	ContentType string
	observer    *observable.Observable
	beforeFunc  func()
	onFunc      OnFunc
}

type OnFunc func(args ...interface{}) ([]byte, error)

var DefaultContentType = "application/json"
var OnSerializableFunc = func(args ...interface{}) ([]byte, error) {
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

func NewEventStream(ob *observable.Observable, event ...string) *EventStream {
	e := &EventStream{
		Events:   event,
		observer: ob,
	}
	return e
}

func (s *EventStream) SetContentType(ct string) {
	s.ContentType = ct
}

func (s *EventStream) Before(beforeFunc func()) {
	s.beforeFunc = beforeFunc
}

func (s *EventStream) On(onFunc OnFunc) {
	s.onFunc = onFunc
}

func (s *EventStream) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.Run(w, r)
}

func (s *EventStream) Run(w http.ResponseWriter, r *http.Request) {
	s.Start(w, r)()
}

func (s *EventStream) Start(w http.ResponseWriter, r *http.Request) func() {

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return func() {}
	}

	event := strings.Join(s.Events, " ")
	msg := make(chan []byte)
	stop := make(chan struct{})

	onFunc := func(args ...interface{}) {
		fn := s.onFunc
		if fn == nil {
			fn = OnSerializableFunc
		}

		var (
			payload []byte
			err     error
		)

		if len(args) > 1 {
			payload, err = fn(args...)
		} else {
			var as []interface{}
			as = append(as, event)
			as = append(as, args...)
			payload, err = fn(as...)
		}

		if err != nil {
			//TODO(anarcher): We need error representation (e.g. HTTPError type?)
			msg <- []byte(fmt.Sprintf("{ \"err\": \"%s\"}", err.Error()))
		}
		select {
		case msg <- payload:
		case <-stop:
			return
		}
	}
	s.observer.On(event, onFunc)

	if s.beforeFunc != nil {
		go s.beforeFunc()
	}

	if s.ContentType == "" {
		s.ContentType = DefaultContentType
	}
	w.Header().Set("Content-Type", s.ContentType)

	return func() {
		defer s.observer.Off(event, onFunc)

		for {
			select {
			case payload := <-msg:
				fmt.Fprintf(w, "%s\n", payload)
				flusher.Flush()
			case <-r.Context().Done():
				close(stop)
				return
			}
		}
	}
}
