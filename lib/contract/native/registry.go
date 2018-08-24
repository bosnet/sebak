package native

var (
	contracts = make(map[string]Register)
)

type (
	Register func(executor *NativeExecutor)
)

func AddContract(addr string, r Register) {
	contracts[addr] = r
}

func HasContract(addr string) bool {
	if _, ok := contracts[addr]; ok {
		return true
	}

	return false
}
