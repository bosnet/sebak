package util

import (
	"bytes"
	"errors"
	"net/url"
	"os"
	"sync"
	"sync/atomic"
	"time"

	uuid "github.com/satori/go.uuid"
)

func NowISO8601() string {
	return time.Now().Format("2006-01-02T15:04:05.000000000Z07:00")
}

func GetUniqueIDFromUUID() string {
	return uuid.Must(uuid.NewV1(), nil).String()
}

func GenerateUUID() string {
	return uuid.Must(uuid.NewV4(), nil).String()
}

func GetUniqueIDFromDate() string {
	return NowISO8601()
}

type CheckerFunc func(target interface{}, args ...interface{}) error

func Checker(checkFuncs ...CheckerFunc) func(interface{}, ...interface{}) error {
	return func(target interface{}, args ...interface{}) (err error) {
		for _, f := range checkFuncs {
			if err := f(target, args...); err != nil {
				return err
			}
		}

		return
	}
}

type SafeLock struct {
	lock  sync.Mutex
	locks int64
}

func (l *SafeLock) Lock() {
	if l.locks < 1 {
		l.lock.Lock()
	}
	atomic.AddInt64(&l.locks, 1)

	return
}

func (l *SafeLock) Unlock() {
	atomic.AddInt64(&l.locks, -1)
	if l.locks < 1 {
		l.lock.Unlock()
	}

	return
}

func GetENVValue(key, defaultValue string) (v string) {
	var found bool
	if v, found = os.LookupEnv(key); !found {
		return defaultValue
	}

	return
}

type SliceFlags []interface{}

func (s *SliceFlags) String() string {
	return "slice flags"
}

func (s *SliceFlags) Set(v string) error {
	if len(v) < 1 {
		return errors.New("empty string found")
	}

	*s = append(*s, v)
	return nil
}

func StripZero(b []byte) []byte {
	var n int
	if n = bytes.Index(b, []byte("\x00")); n != -1 {
		b = b[:n]
	}
	if n = bytes.LastIndex(b, []byte("\x00")); n != -1 {
		b = b[n+1:]
	}

	return b
}

func GetUrlQuery(query url.Values, key, defaultValue string) string {
	v := query.Get(key)
	if len(v) > 0 {
		return v
	}

	return defaultValue
}
