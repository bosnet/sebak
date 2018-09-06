package common

import "fmt"

type CheckerStop interface {
	Error() string
	Checker() Checker
}

type CheckerErrorStop struct {
	checker Checker
	message string
}

func NewCheckerErrorStop(checker Checker, message string) CheckerErrorStop {
	return CheckerErrorStop{
		checker: checker,
		message: message,
	}
}

func (c CheckerErrorStop) Error() string {
	return fmt.Sprintf("stop checker and return: %s", c.message)
}

func (c CheckerErrorStop) Checker() Checker {
	return c.checker
}

type Checker interface {
	GetFuncs() []CheckerFunc
}

type CheckerDeferFunc func(int, Checker, error)

var DefaultDeferFunc CheckerDeferFunc = func(int, Checker, error) {}

type CheckerFunc func(Checker, ...interface{}) error

type DefaultChecker struct {
	Funcs []CheckerFunc
}

func (c *DefaultChecker) GetFuncs() []CheckerFunc {
	return c.Funcs
}

func RunChecker(checker Checker, deferFunc CheckerDeferFunc, args ...interface{}) error {
	if deferFunc == nil {
		deferFunc = DefaultDeferFunc
	}

	var err error
	for i, f := range checker.GetFuncs() {
		if err = f(checker, args...); err != nil {
			deferFunc(i, checker, err)
			return err
		}
		deferFunc(i, checker, err)
	}
	return nil
}
