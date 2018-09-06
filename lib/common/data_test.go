package common

import (
	"encoding/json"
	"testing"
)

type A struct {
	A string
	B B
}

type B struct {
	B string
}

func (b B) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{"b": b.B, "c": "findme"})
}

func TestJSONMarshalJSONFunc(t *testing.T) {
	a := A{A: "a", B: B{B: "b"}}
	m, err := json.Marshal(a)
	if err != nil {
		t.Error(err)
		return
	}

	var d map[string]interface{}
	if err = json.Unmarshal(m, &d); err != nil {
		t.Error(err)
		return
	}
	n := d["B"].(map[string]interface{})

	if v, ok := n["c"]; !ok {
		t.Error("failed to find 'c'")
		return
	} else if w, ok := v.(string); !ok {
		t.Error("failed to find 'c'")
		return
	} else if w != "findme" {
		t.Error("failed to find 'c'")
		return
	}
}
