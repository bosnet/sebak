package jsvm

import (
	"boscoin.io/sebak/lib/contract/api"
	"boscoin.io/sebak/lib/contract/value"
	"github.com/robertkrimen/otto"
)

func HelloWorldFunc(api *api.API) func(call otto.FunctionCall) otto.Value {
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

func SetStatusFunc(api *api.API) func(call otto.FunctionCall) otto.Value {
	return func(call otto.FunctionCall) otto.Value {
		name, err := call.Argument(0).ToString()
		if err != nil {
			return otto.FalseValue()
		}

		v, err := value.ToValue(call.Argument(1))
		if err != nil {
			return otto.FalseValue()
		}

		err = api.PutStorageItem([]byte(name), v)
		if err != nil {
			return otto.FalseValue()
		}

		return otto.TrueValue()
	}
}

func GetStatusFunc(api *api.API) func(call otto.FunctionCall) otto.Value {
	return func(call otto.FunctionCall) otto.Value {
		name, err := call.Argument(0).ToString()
		if err != nil {
			return otto.NullValue()
		}

		v, err := api.GetStorageItem([]byte(name))
		if err != nil {
			return otto.NullValue()
		}

		ottoValue, err := otto.ToValue(v.Value())
		if err != nil {
			return otto.NullValue()
		}

		return ottoValue
	}
}
