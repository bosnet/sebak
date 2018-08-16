package jsvm

import (
	"boscoin.io/sebak/lib/contract/api"
	"boscoin.io/sebak/lib/contract/payload"
	"github.com/robertkrimen/otto"
)

func HelloWorldFunc(api api.API) func(otto.FunctionCall) otto.Value {
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

func CallContractFunc(api api.API) func(otto.FunctionCall) otto.Value {
	return func(call otto.FunctionCall) otto.Value {
		address, err := call.Argument(0).ToString()
		if err != nil {
			panic(err)
		}
		method, err := call.Argument(1).ToString()
		if err != nil {
			panic(err)
		}

		var args []string
		for _, a := range call.ArgumentList[1:] {
			arg, err := a.ToString()
			if err != nil {
				panic(err)
			}

			args = append(args, arg) // TODO:
		}

		execCode := &payload.ExecCode{
			ContractAddress: address,
			Method:          method,
			Args:            args,
		}

		retCode, err := api.CallContract(execCode)
		if err != nil {
			panic(err)
		}

		ret, err := otto.ToValue(string(retCode.Contents)) //TODO: It is a better if value.Value.Contents is `interface{}` ?
		if err != nil {
			panic(err)
		}

		return ret
	}
}
