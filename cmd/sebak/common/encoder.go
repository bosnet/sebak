package common

import (
	"io"
	"encoding/json"
	"gopkg.in/yaml.v2"
)

type Encode func(v interface{}, w io.Writer) error

var DefaultEncodes = map[string]Encode{
	"json": func(v interface{}, w io.Writer) error {
		return jsonEncode(v, w, false)
	},
	"prettyjson": func(v interface{}, w io.Writer) error {
		return jsonEncode(v, w, true)
	},
	"yaml": func(v interface{}, w io.Writer) error {
		e := yaml.NewEncoder(w)
		return e.Encode(v)
	},
}

func jsonEncode(v interface{}, w io.Writer, pretty bool) error {
	e := json.NewEncoder(w)
	if pretty {
		e.SetIndent("", "  ")
	}

	return e.Encode(&v)
}
