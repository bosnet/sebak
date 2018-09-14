package runner

import (
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/transaction"
)

type Issuance struct {
	idContract string
	hStart     uint64
	hEnd       uint64
	hInterval  uint64
	issueUnit  common.Amount
	issueTotal common.Amount
	budget     string
}

func NewIssuance(hash string, start uint64, end uint64, interval uint64, unit common.Amount, total common.Amount, address string) *Issuance {
	return &Issuance{
		idContract: hash,
		hStart:     start,
		hEnd:       end,
		hInterval:  interval,
		issueUnit:  unit,
		issueTotal: total,
		budget:     address,
	}
}

func NewIssuanceByContract(hash string) *Issuance {
	var i *Issuance
	// read a recardian contract and config the issuance from it.

	return i
}

func (i *Issuance) IsAvailableIssuance(height uint64) (available bool) {
	available = true
	if i.hEnd < height {
		available = false
	}
	return
}

func (i *Issuance) GetHash() string {
	return i.idContract
}

func (i *Issuance) Issue(height uint64) (op transaction.Operation, available bool) {
	available = true

	if (i.hEnd < height) || ((height-i.hStart)%i.hInterval) != 0 {
		available = false
		return
	}
	opb := transaction.NewOperationBodyIssuance(i.budget, i.issueUnit)

	op = transaction.Operation{
		H: transaction.OperationHeader{
			Type: transaction.OperationIssuance,
		},
		B: opb,
	}
	return
}

type IssuancePool struct {
	Pool   map[ /* Issuance.GetHash() */ string]Issuance
	Hashes []string /* Issuance.GetHash() */
}

func NewIssuancePool() *IssuancePool {
	return &IssuancePool{
		Pool:   map[string]Issuance{},
		Hashes: []string{},
	}
	// and add the default inflation for common budget

}
func (ip *IssuancePool) Len() int {
	return len(ip.Hashes)
}

func (ip *IssuancePool) Has(hash string) bool {
	_, found := ip.Pool[hash]
	return found
}

func (ip *IssuancePool) Get(hash string) (i Issuance, found bool) {
	i, found = ip.Pool[hash]
	return
}

func (ip *IssuancePool) Add(i Issuance) bool {
	if _, found := ip.Pool[i.GetHash()]; found {
		return false
	}

	ip.Pool[i.GetHash()] = i
	ip.Hashes = append(ip.Hashes, i.GetHash())

	return true
}

func (ip *IssuancePool) Remove(hash string) {
	if len(hash) < 1 || ip.Has(hash) {
		return
	}
	delete(ip.Pool, hash)

	index, found := common.InStringArray(ip.Hashes, hash)
	if found {
		ip.Hashes[index] = ip.Hashes[len(ip.Hashes)-1]
		ip.Hashes[len(ip.Hashes)-1] = ""
		ip.Hashes = ip.Hashes[:len(ip.Hashes)-1]
	}
}

func (ip *IssuancePool) Validate(height uint64, tx transaction.Transaction) (valid bool) {
	return
}

func (ip *IssuancePool) Issue(height uint64) (tx transaction.Transaction, available bool) {
	var ops []transaction.Operation

	for _, hash := range ip.Hashes {
		issuance := ip.Pool[hash]
		if issuance.IsAvailableIssuance(height) {
			op, opavail := issuance.Issue(height)
			if opavail {
				ops = append(ops, op)
			}
		} else {
			ip.Remove(hash)
		}
	}
	if len(ops) > 0 {
		available = true
		txBody := transaction.TransactionBody{
			Source:     "",
			Fee:        common.Amount(0),
			SequenceID: 0,
			Operations: ops,
		}

		tx = transaction.Transaction{
			T: "transaction-ex",
			H: transaction.TransactionHeader{
				Created: common.NowISO8601(),
				Hash:    txBody.MakeHashString(),
			},
			B: txBody,
		}
	} else {
		available = false
	}
	return
}
