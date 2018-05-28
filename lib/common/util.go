package sebakcommon

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
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

type CheckerErrorStop struct {
	Message string
}

func (c CheckerErrorStop) Error() string {
	return fmt.Sprintf("stop checker and return: %s", c.Message)
}

type CheckerFunc func(context.Context, interface{}, ...interface{}) (context.Context, error)
type DeferFunc func(int, CheckerFunc, context.Context, error)

func Checker(ctx context.Context, checkFuncs ...CheckerFunc) func(interface{}, ...interface{}) (context.Context, error) {
	deferFunc := func(int, CheckerFunc, context.Context, error) {}

	if ctx != nil {
		if deferFuncValue := ctx.Value("deferFunc"); deferFuncValue != nil {
			deferFunc = deferFuncValue.(DeferFunc)
		}
	}

	return func(target interface{}, args ...interface{}) (context.Context, error) {
		for i, f := range checkFuncs {
			var err error
			if ctx, err = f(ctx, target, args...); err != nil {
				deferFunc(i, f, ctx, err)
				return ctx, err
			}
			deferFunc(i, f, ctx, err)
		}
		return ctx, nil
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

func InTestVerbose() bool {
	flag.Parse()
	if v := flag.Lookup("test.v"); v == nil || v.Value.String() != "true" {
		return false
	}

	return true
}

func InTest() bool {
	flag.Parse()
	if v := flag.Lookup("test.v"); v == nil {
		return false
	}

	return true
}

func InStringArray(a []string, s string) (index int, found bool) {
	var h string
	for index, h = range a {
		found = h == s
		if found {
			return
		}
	}

	index = -1
	return
}

func MustJSONMarshal(o interface{}) []byte {
	b, _ := json.Marshal(o)
	return b
}

func ReverseStringSlice(a []string) []string {
	if len(a) < 1 {
		return []string{}
	}
	b := make([]string, len(a))
	for i, j := 0, len(a)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = a[j], a[i]
	}
	return b
}
