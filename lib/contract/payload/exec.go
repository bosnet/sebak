package payload

import "boscoin.io/sebak/lib/common"

type ExecCode struct {
	ContractAddress string
	Method          string
	Args            []string
}

func (ec *ExecCode) Serialize() (encoded []byte, err error) {
	encoded, err = sebakcommon.EncodeJSONValue(ec)
	return
}

func (ec *ExecCode) Deserialize(encoded []byte) (err error) {
	if ec == nil {
		ec = new(ExecCode)
	}
	err = sebakcommon.DecodeJSONValue(encoded, ec)
	return
}
