package jsvm

import (
	"boscoin.io/sebak/lib/contract/api"
	"github.com/robertkrimen/otto"
)

func HelloWorldFunc(api api.API) func(call otto.FunctionCall) otto.Value {
	return func(call otto.FunctionCall) otto.Value {
		greeting, err := call.Argument(0).ToString()
		if err != nil {
			panic(err)
		}
		retStr, err := api.Helloworld(greeting)
		if err != nil {
			panic(err)
		}

		retValue, err := otto.ToValue(retStr)
		if err != nil {
			panic(err)
		}
		return retValue
	}
}
