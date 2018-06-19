package sebakcommon

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/btcsuite/btcutil/base58"
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

func IsExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func IsNotExists(path string) bool {
	_, err := os.Stat(path)
	return os.IsNotExist(err)
}

func IsEmpty(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1) // Or f.Readdir(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err // Either not empty or error, suits both cases
}

func MakeCheckpoint(a, b string) string {
	return fmt.Sprintf("%s-%s", a, b)
}

func ParseCheckpoint(a string) (p [2]string, err error) {
	s := strings.SplitN(a, "-", 2)
	if len(s) != 2 {
		err = errors.New("invalid checkpoint")
		return
	}
	p = [2]string{s[0], s[1]}
	return
}

func MakeGenesisCheckpoint(networkID []byte) string {
	h := base58.Encode(networkID)
	return MakeCheckpoint(h, h)
}
