package sebakcommon

import (
	"fmt"
	"sort"
	"testing"

	"github.com/satori/go.uuid"
)

func TestSequentialUUIDWithSatori(t *testing.T) {
	var ids []string
	for i := 0; i < 500000; i++ {
		ids = append(ids, uuid.Must(uuid.NewV1(), nil).String())
	}

	sortedIds := make([]string, len(ids))
	copy(sortedIds, ids)
	sort.Strings(sortedIds)

	for i, id := range ids {
		if id != sortedIds[i] {
			t.Error("failed to make sequential id thru `satori/go-uuid`")
			return
		}
	}
}

func TestInStringArray(t *testing.T) {
	var ids []string
	for i := 0; i < 3; i++ {
		ids = append(
			ids,
			fmt.Sprintf("%s-%s", uuid.Must(uuid.NewV1(), nil).String(), uuid.Must(uuid.NewV1(), nil).String()),
		)
	}

	if index, found := InStringArray(ids, ids[0]); index != 0 && !found {
		t.Error("failed to search", ids, ids[0])
		return
	}
	if index, found := InStringArray(ids, ids[len(ids)-1]); index != len(ids)-1 && !found {
		t.Error("failed to search", ids, ids[len(ids)-1])
		return
	}
	if index, found := InStringArray(ids, ids[2]); index != 2 && !found {
		t.Error("failed to search", ids, ids[2])
		return
	}
	if index, found := InStringArray(ids, "findme"); index != -1 && found {
		t.Error("failed to search", ids, "findme")
		return
	}

	as := []string{
		"GCZBKG5ZNBDJ4E46JSCEU6AABJ6ZFQRKXL5B7JOUEGPKLTSI545VHO7B",
		"GCHXRPJLWOFFZKUPJUO7LMGKRGTGASZJCOXB32XPPHJWG6PQNHORFYYE1",
	}
	s := "GCZBKG5ZNBDJ4E46JSCEU6AABJ6ZFQRKXL5B7JOUEGPKLTSI545VHO7B"
	if index, found := InStringArray(as, s); index != 0 && !found {
		t.Error("failed to search", as, s)
		return
	}
}
