package common

import (
	"io"
	"encoding/json"
	"fmt"
)

type (
	Encoder interface {
		Encode(w io.Writer) error
	}

	EncodeFn func(v interface{}) Encoder
	EncodeFnMap map[string]EncodeFn
	Encoders struct {
		encoders EncodeFnMap
	}
)

var (
	predefined = EncodeFnMap{
		"json": func(v interface{}) Encoder {
			fmt.Println("O?")
			return &jsonEncoder{v, false}
		},
		"prettyjson": func(v interface{}) Encoder {
			return &jsonEncoder{v, true}
		},
		// TODO: YAML
	}
)

func NewEncoders(extra EncodeFnMap) Encoders {
	if len(extra) == 0 {
		return Encoders{
			encoders: predefined,
		}
	} else {
		newEncoders := make(map[string]EncodeFn)

		for _, fns := range [2]EncodeFnMap{predefined, extra} {
			for format, fn := range fns {
				newEncoders[format] = fn
			}
		}

		return Encoders{
			encoders: newEncoders,
		}
	}
}

func (o *Encoders) Get(format string) (fn EncodeFn, ok bool) {
	fn, ok = o.encoders[format]
	return
}

type jsonEncoder struct {
	v      interface{}
	pretty bool
}

func (o *jsonEncoder) Encode(w io.Writer) error {
	je := json.NewEncoder(w)
	if o.pretty {
		je.SetIndent("", "  ")
	}
	return je.Encode(&o.v)
}
