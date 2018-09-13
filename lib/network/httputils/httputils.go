package httputils

import (
	"boscoin.io/sebak/lib/error"
	"net/http"
)

// IsEventStream checks request header accept is text/event-stream
func IsEventStream(r *http.Request) bool {
	if r.Header.Get("Accept") == "text/event-stream" {
		return true

	}
	return false
}

var (
	ErrorsToStatus = map[uint]int{
		//TODO: set relevant code
		100: 400,
		101: 400,
		102: 400,
		103: 400,
		104: 400,
		105: 400,
		106: 400,
		107: 400,
		108: 400,
		109: 400,
		110: 400,
		111: 400,
		112: 400,
		113: 400,
		114: 400,
		115: 400,
		116: 400,
		118: 400,
		119: 400,
		120: 400,
		121: 400,
		122: 400,
		123: 400,
		124: 400,
		125: 400,
		126: 400,
		127: 400,
		128: 400,
		129: 400,
		130: 400,
		131: 400,
		132: 400,
		133: 400,
		134: 400,
		135: 400,
		136: 400,
		137: 400,
		138: 400,
		139: 400,
		140: 400,
		141: 400,
		142: 400,
		143: 400,
		144: 400,
		145: 400,
	}
)

func StatusCode(err error) int {
	if e, ok := err.(*errors.Error); ok {
		return ErrorsToStatus[e.Code]
	}
	return 500
}
