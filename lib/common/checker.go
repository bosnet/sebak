package sebakcommon

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
