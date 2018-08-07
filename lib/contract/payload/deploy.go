package payload

type CodeType int

const (
	Native CodeType = iota
	JavaScript
	WASM
	NONE
)

type DeployCode struct {
	ContractAddress string
	Type            CodeType // wasm,otto,...
	Code            []byte
}

func (dc *DeployCode) Serialize() (encoded []byte, err error) {
	encoded, err = EncodeJSONValue(dc)
	return
}

func (dc *DeployCode) Deserialize(encoded []byte) (err error) {
	if dc == nil {
		dc = new(DeployCode)
	}
	err = DecodeJSONValue(encoded, dc)
	return
}
