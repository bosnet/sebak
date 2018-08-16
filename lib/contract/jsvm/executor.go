package jsvm

import (
	"log"
	"regexp"

	"boscoin.io/sebak/lib/contract/api"
	"boscoin.io/sebak/lib/contract/context"
	"boscoin.io/sebak/lib/contract/payload"
	"boscoin.io/sebak/lib/contract/value"

	"github.com/robertkrimen/otto"
)

var r, _ = regexp.Compile("^[A-Z]")

var intrinsicFunctions = []string{
	"Error",
	"URIError",
	"RangeError",
	"SyntaxError",
	"EvalError",
	"TypeError",
	"ReferenceError",
	"Date",
	"Number",
	"RegExp",
	"String",
	"Boolean",
	"Object",
	"Function",
	"Array",
}

type OttoExecutor struct {
	Context   context.Context
	api       api.API
	functions map[string]otto.Value
	VM        *otto.Otto
}

func NewOttoExecutor(ctx context.Context, api api.API, deployCode *payload.DeployCode) *OttoExecutor {

	vm := otto.New()
	vm.Run(deployCode.Code)

	functions := make(map[string]otto.Value)
	for key, value := range vm.Context().Symbols {
		if value.IsFunction() && r.MatchString(key) && !contains(intrinsicFunctions, key) {
			functions[key] = value
		}
	}

	ex := &OttoExecutor{
		Context:   ctx,
		api:       api,
		functions: functions,
		VM:        vm,
	}

	ex.RegisterFuncs()

	return ex
}

func (ex *OttoExecutor) Execute(c *payload.ExecCode) (*value.Value, error) {

	function := ex.functions[c.Method]
	ivalue, err := function.Call(function, c.Args)
	if err != nil {
		if ottoError, ok := err.(*otto.Error); ok {
			log.Println(ottoError.String()) // line number from: https://github.com/robertkrimen/otto/issues/265
		}
		return nil, err
	}

	retCode, err := value.ToValue(ivalue)
	if err != nil {
		return nil, err
	}
	return retCode, nil
}

func (ex *OttoExecutor) RegisterFuncs() {
	ex.VM.Set("HelloWorld", HelloWorldFunc(ex.api))
	ex.VM.Set("CallContract", CallContractFunc(ex.api))
}

func contains(slice []string, item string) bool {
	set := make(map[string]struct{}, len(slice))
	for _, s := range slice {
		set[s] = struct{}{}
	}

	_, ok := set[item]
	return ok
}
