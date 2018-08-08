package wasm

import (
	"boscoin.io/sebak/lib/contract/context"
	"boscoin.io/sebak/lib/contract/payload"
	"boscoin.io/sebak/lib/contract/value"

	"github.com/go-interpreter/wagon/exec"
	"github.com/go-interpreter/wagon/validate"
	"github.com/go-interpreter/wagon/wasm"

	"os"
)

type ExecFunc func(e *WasmExecutor, code *payload.ExecCode) (*value.Value, error)

type WasmExecutor struct {
	Context *context.Context
	//Engine  *exec.ExecutionEngine
}

func NewWasmExecutor(ctx *context.Context) *WasmExecutor {
	ex := &WasmExecutor{
		Context: ctx,
	}

	return ex
}

//TODO: it is just a dummy
func (ex *WasmExecutor) Execute(c *payload.ExecCode) (retCode *value.Value, err error) {
	file, err := os.Open("dummy")
	if err != nil {
		return
	}
	defer file.Close()

	module, err := wasm.ReadModule(file, nil)
	if err != nil {
		return
	}
	if err = validate.VerifyModule(module); err != nil {
		return
	}

	exec.NewVM(module)
	//
	//engine := exec.NewExecutionEngine(nil, "product")
	//
	////TODO: set wasmcode and input
	//var wasmcode []byte //wasm byte code
	//var input []byte    //method | arguments...
	//var result value.Value
	//
	//{
	//	//TODO: encode input to byte
	//}
	//res, err := engine.Call(nil, wasmcode, input)
	//if err != nil {
	//	return nil, err
	//}
	//
	//{
	//	//TODO: decode return byte to ReturnCode
	//	_, err = engine.GetVM().GetPointerMemory(uint64(binary.LittleEndian.Uint32(res)))
	//	if err != nil {
	//		return nil, err
	//	}
	//}
	//
	//return &result, err
	return
}
