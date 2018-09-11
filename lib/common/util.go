package common

import (
	"bytes"
	"encoding/json"
	"io"
	"net/url"
	"os"
	"sync"
	"sync/atomic"

	"encoding/binary"
	uuid "github.com/satori/go.uuid"
	"github.com/stellar/go/keypair"
)

const MaxUintEncodeByte = 8

func GetUniqueIDFromUUID() string {
	return uuid.Must(uuid.NewV1(), nil).String()
}

func GenerateUUID() string {
	return uuid.Must(uuid.NewV4(), nil).String()
}

func GetUniqueIDFromDate() string {
	return NowISO8601()
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

func GetUrlQuery(query url.Values, key, defaultValue string) string {
	v := query.Get(key)
	if len(v) > 0 {
		return v
	}

	return defaultValue
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

func InStringMap(a map[string]bool, s string) (found bool) {
	_, found = a[s]
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

func IsStringArrayEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}

	return true
}

func IsStringMapEqual(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for hash := range a {
		if _, ok := b[hash]; !ok {
			return false
		}
	}

	return true
}

func IsStringMapEqualWithHash(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	aHash := MustMakeObjectHash(a)
	bHash := MustMakeObjectHash(b)

	return bytes.Equal(aHash, bHash)
}

// MakeSignature makes signature from given hash string
func MakeSignature(kp keypair.KP, networkID []byte, hash string) ([]byte, error) {
	return kp.Sign(append(networkID, []byte(hash)...))
}

func EncodeUint64ToByteSlice(i uint64) [MaxUintEncodeByte]byte {
	var b [MaxUintEncodeByte]byte
	binary.BigEndian.PutUint64(b[:], i)
	return b
}
