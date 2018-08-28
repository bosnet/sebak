package execfunc

import (
	"boscoin.io/sebak/lib/contract/native"
	"boscoin.io/sebak/lib/contract/payload"
	"boscoin.io/sebak/lib/contract/value"
)

var HelloWorldAddress = "HELLOWORLDADDRESS"

func init() {
	native.AddContract(HelloWorldAddress, RegisterHelloWorld)
}

func RegisterHelloWorld(ex *native.NativeExecutor) {
	ex.RegisterFunc("Hello", hello)
}

func hello(ex *native.NativeExecutor, execCode *payload.ExecCode) (ret *value.Value, err error) {
	greeting := execCode.Args[0]
	retHello, err := ex.API().Helloworld(greeting.(string))

	ret, _ = value.ToValue(retHello)
	return
}
