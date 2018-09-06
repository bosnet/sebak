package common

import (
	"errors"
	"testing"
)

func TestChecker(t *testing.T) {
	limit := 10
	funcs := []CheckerFunc{}
	var dones []interface{}
	for i := 0; i < limit; i++ {
		f := func(checker Checker, args ...interface{}) error {
			dones = append(dones, checker)
			return nil
		}
		funcs = append(funcs, f)
	}

	checker := &DefaultChecker{funcs}
	err := RunChecker(checker, DefaultDeferFunc)
	if err != nil {
		t.Error(err)
		return
	}

	if len(dones) != limit {
		t.Error("some funcs were not executed")
		return
	}
}

type CheckerWithProperties struct {
	DefaultChecker

	P0 int
}

func TestCheckerWithProperties(t *testing.T) {
	funcs := []CheckerFunc{}
	f0 := func(c Checker, args ...interface{}) error {
		checker := c.(*CheckerWithProperties)
		checker.P0 = 99
		return nil
	}
	funcs = append(funcs, f0)

	f1 := func(c Checker, args ...interface{}) error {
		checker := c.(*CheckerWithProperties)
		if checker.P0 != 99 {
			err := errors.New("failed to set property in Checker")
			t.Error(err)
			return err
		}
		return nil
	}
	funcs = append(funcs, f1)

	checker := &CheckerWithProperties{DefaultChecker: DefaultChecker{funcs}}
	err := RunChecker(checker, DefaultDeferFunc)
	if err != nil {
		t.Error(err)
		return
	}
}
