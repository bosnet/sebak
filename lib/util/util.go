package util

import (
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
