package payload

type ExecCode struct {
	ContractAddress string
	Method          string
	Args            []string
}

func (ec *ExecCode) Serialize() (encoded []byte, err error) {
	encoded, err = EncodeJSONValue(ec)
	return
}

func (ec *ExecCode) Deserialize(encoded []byte) (err error) {
	if ec == nil {
		ec = new(ExecCode)
	}
	err = DecodeJSONValue(encoded, ec)
	return
}
