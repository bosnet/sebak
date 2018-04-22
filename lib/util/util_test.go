package util

import (
	"sort"
	"testing"

	"github.com/satori/go.uuid"
)

func TestSequentialUUIDWithsatori(t *testing.T) {
	var ids []string
	for i := 0; i < 500000; i++ {
		ids = append(ids, uuid.Must(uuid.NewV1()).String())
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
