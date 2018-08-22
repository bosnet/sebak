package payload

import (
	"boscoin.io/sebak/lib/storage"

	"boscoin.io/sebak/lib/common"
	"fmt"
)

type CodeType int

const (
	Native CodeType = iota
	JavaScript
	WASM
	NONE
)

const (
	DeployCodePrefixAddress string = "tc-address-"
)

type DeployCode struct {
	ContractAddress string
	Type            CodeType // wasm,otto,...
	Code            []byte
}

func (dc *DeployCode) Save(st sebakstorage.DBBackend) error {
	return st.New(GetDeployCodeDBKey(dc.ContractAddress), dc)
}

func (dc *DeployCode) Serialize() (encoded []byte, err error) {
	encoded, err = sebakcommon.EncodeJSONValue(dc)
	return
}

func (dc *DeployCode) Deserialize(encoded []byte) (err error) {
	if dc == nil {
		dc = new(DeployCode)
	}
	err = sebakcommon.DecodeJSONValue(encoded, dc)
	return
}

func GetDeployCodeDBKey(address string) string {
	return fmt.Sprintf("%s%s", DeployCodePrefixAddress, address)
}

func GetDeployCode(st sebakstorage.DBBackend, addr string) (*DeployCode, error) {
	var dc *DeployCode
	if err := st.Get(GetDeployCodeDBKey(addr), &dc); err != nil {
		return nil, err
	}
	return dc, nil

}
