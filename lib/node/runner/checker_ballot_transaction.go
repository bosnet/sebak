package runner

import (
	"strconv"
	"strings"
	"sync"

	"boscoin.io/sebak/lib/ballot"
	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/errors"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/storage"
	"boscoin.io/sebak/lib/transaction"
	"boscoin.io/sebak/lib/transaction/operation"
	"boscoin.io/sebak/lib/voting"
)

type BallotTransactionChecker struct {
	sync.RWMutex
	common.DefaultChecker

	NodeRunner *NodeRunner
	Conf       common.Config
	LocalNode  node.Node
	NetworkID  []byte

	Ballot                ballot.Ballot
	Transactions          []string
	VotingHole            voting.Hole
	ValidTransactions     []string
	validTransactionsMap  map[string]bool
	CheckTransactionsOnly bool
	transactionCache      *TransactionCache
}

func (checker *BallotTransactionChecker) invalidTransactions() (invalids []string) {
	for _, hash := range checker.Transactions {
		if _, found := checker.validTransactionsMap[hash]; found {
			continue
		}

		invalids = append(invalids, hash)
	}

	return
}

func (checker *BallotTransactionChecker) setValidTransactions(hashes []string) {
	checker.ValidTransactions = hashes

	checker.validTransactionsMap = map[string]bool{}
	for _, hash := range hashes {
		checker.validTransactionsMap[hash] = true
	}

	return
}

// IsNew checks the incoming transaction is
// already stored or not.
func IsNew(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*BallotTransactionChecker)

	var validTransactions []string
	for _, hash := range checker.Transactions {
		// check transaction is already stored
		var found bool
		if found, err = block.ExistsBlockTransaction(checker.NodeRunner.Storage(), hash); err != nil || found {
			if !checker.CheckTransactionsOnly {
				err = errors.NewButKnownMessage
				return
			}
			continue
		}
		validTransactions = append(validTransactions, hash)
	}

	err = nil
	checker.setValidTransactions(validTransactions)

	return
}

// CheckMissingTransaction will get the missing tranactions, that is, not in
// `Pool` from proposer.
func CheckMissingTransaction(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*BallotTransactionChecker)

	var found bool
	var validTransactions []string
	for _, hash := range checker.ValidTransactions {
		if _, found, err = checker.transactionCache.Get(hash); err != nil {
			return
		} else if !found {
			continue
		}
		validTransactions = append(validTransactions, hash)
	}

	checker.setValidTransactions(validTransactions)

	return
}

// BallotTransactionsOperationLimit checks the total number of operations is
// over `OpsInBallotLimit`.
func BallotTransactionsOperationLimit(c common.Checker, args ...interface{}) error {
	checker := c.(*BallotTransactionChecker)

	var ops int
	for _, hash := range checker.ValidTransactions {
		if tx, found, err := checker.transactionCache.Get(hash); err != nil {
			return err
		} else if !found {
			return errors.TransactionNotFound
		} else {
			ops += len(tx.B.Operations)
			if ops > checker.Conf.OpsInBallotLimit {
				return errors.BallotHasOverMaxOperationsInBallot
			}
		}
	}

	return nil
}

// BallotTransactionsSameSource checks there are transactions which has same
// source in the `Transactions`.
func BallotTransactionsSameSource(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*BallotTransactionChecker)

	var validTransactions []string
	sources := map[string]bool{}

	var tx transaction.Transaction
	var found bool
	for _, hash := range checker.ValidTransactions {
		if tx, found, err = checker.transactionCache.Get(hash); err != nil {
			return
		} else if !found {
			continue
		}

		if found := common.InStringMap(sources, tx.B.Source); found {
			if !checker.CheckTransactionsOnly {
				err = errors.TransactionSameSourceInBallot
				return
			}
			continue
		}

		sources[tx.B.Source] = true
		validTransactions = append(validTransactions, hash)
	}
	err = nil
	checker.setValidTransactions(validTransactions)

	return
}

// BallotTransactionsOperationBodyCollectTxFee validates the
// `BallotTransactionsOperationBodyCollectTxFee.Amount` is matched with the
// collected fee of all transactions.
func BallotTransactionsOperationBodyCollectTxFee(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*BallotTransactionChecker)

	var opb operation.CollectTxFee
	if opb, err = checker.Ballot.ProposerTransaction().CollectTxFee(); err != nil {
		return
	}

	// check the colleted transaction fee is matched with
	// `CollectTxFee.Amount`
	if checker.Ballot.TransactionsLength() < 1 {
		if opb.Amount != 0 {
			err = errors.InvalidOperation
			return
		}
	} else {
		var fee common.Amount
		var tx transaction.Transaction
		var found bool
		for _, hash := range checker.Transactions {
			if tx, found, err = checker.transactionCache.Get(hash); err != nil {
				return
			} else if !found {
				err = errors.TransactionNotFound
				return
			}
			fee = fee.MustAdd(tx.B.Fee)
		}
		if opb.Amount != fee {
			err = errors.InvalidFee
			return
		}
	}

	return
}

// BallotTransactionsAllValid checks all the transactions are valid or not.
func BallotTransactionsAllValid(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*BallotTransactionChecker)

	if len(checker.InvalidTransactions()) > 0 {
		checker.VotingHole = voting.NO
	} else {
		checker.VotingHole = voting.YES
	}

	return
}

//
// Validate the entirety of a transaction
//
// This function is critical for consensus, as it defines the validation rules for a transaction.
// It is executed whenever a transaction is received.
//
// As it is run on every single transaction, it should be as fast as possible,
// thus any new code should carefully consider when it takes place.
// Consider putting cheap, frequent errors first.
//
// Note that if we get to this function, it means the transaction is known to be
// well-formed (e.g. the signature is legit).
//
// Params:
//   st = Storage backend to use (e.g. to access the blocks)
//        Only ever read from, never written to.
//   config = consist of configuration of the network. common address, congress address, etc.
//   tx = Transaction to check
//
func ValidateTx(st *storage.LevelDBBackend, config common.Config, tx transaction.Transaction) (err error) {
	// check, source exists
	var ba *block.BlockAccount
	if ba, err = block.GetBlockAccount(st, tx.B.Source); err != nil {
		return errors.BlockAccountDoesNotExists
	}

	// check, version is correct
	if !tx.IsValidVersion(common.TransactionVersionV1) {
		err = errors.InvalidMessageVersion
		return
	}

	// check, sequenceID is based on latest sequenceID
	if !tx.IsValidSequenceID(ba.SequenceID) {
		err = errors.TransactionInvalidSequenceID
		return
	}

	totalAmount := tx.TotalAmount(true)

	// check, have enough balance at sequenceID
	if ba.Balance < totalAmount {
		err = errors.TransactionExcessAbilityToPay
		return
	}

	for _, op := range tx.B.Operations {
		if err = ValidateOp(st, config, ba, op); err != nil {
			return
		}
	}

	return
}

//
// Validate an operation
//
// This function is critical for consensus, as it defines the validation rules for an operation,
// and by extension a transaction. It is called from ValidateTx.
//
// Params:
//   st = Storage backend to use (e.g. to access the blocks)
//        Only ever read from, never written to.
//   config = consist of configuration of the network. common address, congress address, etc.
//   source = Account from where the transaction (and ops) come from
//   tx = Transaction to check
//
func ValidateOp(st *storage.LevelDBBackend, config common.Config, source *block.BlockAccount, op operation.Operation) (err error) {

	var funcIsFrozenPayable = func(source *block.BlockAccount) (err error) {
		// Unfreezing must be done after X period from unfreezing request
		iterFunc, closeFunc := block.GetBlockOperationsBySource(st, source.Address, nil)
		bo, _, _ := iterFunc() //Get the first operation submitted by the source(frozen) account
		closeFunc()
		// Before unfreezing payment, unfreezing request shoud be saved
		if bo.Type != operation.TypeUnfreezingRequest {
			return errors.UnfreezingRequestNotRequested
		}
		lastblock := block.GetLatestBlock(st)
		// unfreezing period is 241920.
		if lastblock.Height-bo.Height < common.UnfreezingPeriod {
			return errors.UnfreezingNotReachedExpiration
		}
		return nil
	}

	switch op.H.Type {
	case operation.TypeCreateAccount:
		var ok bool
		var casted operation.CreateAccount
		if casted, ok = op.B.(operation.CreateAccount); !ok {
			return errors.TypeOperationBodyNotMatched
		}

		if exists, err := block.ExistsBlockAccount(st, casted.Target); err == nil && exists {
			return errors.BlockAccountAlreadyExists
		}

		if source.IsFrozen() {
			if err = funcIsFrozenPayable(source); err != nil {
				return err
			}
		}

	case operation.TypePayment:
		var ok bool
		var casted operation.Payment
		if casted, ok = op.B.(operation.Payment); !ok {
			return errors.TypeOperationBodyNotMatched
		}
		var taccount *block.BlockAccount
		var err error
		if taccount, err = block.GetBlockAccount(st, casted.Target); err != nil {
			return errors.BlockAccountDoesNotExists
		}

		// If it's a frozen account, it cannot receive payment
		if taccount.IsFrozen() {
			return errors.FrozenAccountNoDeposit
		}

		// The source account is frozen account
		if source.IsFrozen() {
			if err = funcIsFrozenPayable(source); err != nil {
				return err
			}
		}
	case operation.TypeUnfreezingRequest:
		if _, ok := op.B.(operation.UnfreezeRequest); !ok {
			return errors.TypeOperationBodyNotMatched
		}
		// Unfreezing should be done from a frozen account
		if !source.IsFrozen() {
			return errors.UnfreezingFromInvalidAccount
		}
		// Repeated unfreeze request shoud be blocked after unfreeze request saved
		iterFunc, closeFunc := block.GetBlockOperationsBySource(st, source.Address, nil)
		bo, _, _ := iterFunc()
		closeFunc()
		if bo.Type == operation.TypeUnfreezingRequest {
			return errors.UnfreezingRequestAlreadyReceived
		}
	case operation.TypeInflationPF:
		var ok bool
		var inflationPF operation.InflationPF
		if inflationPF, ok = op.B.(operation.InflationPF); !ok {
			return errors.TypeOperationBodyNotMatched
		}
		var taccount *block.BlockAccount
		if taccount, err = block.GetBlockAccount(st, inflationPF.FundingAddress); err != nil {
			return errors.BlockAccountDoesNotExists
		}
		// If it's a frozen account, it cannot receive payment
		if taccount.IsFrozen() {
			return errors.FrozenAccountNoDeposit
		}

		if config.CommonAccountAddress != source.Address {
			return errors.InvalidOperation
		}

		var congressVotingHash string
		{
			var bo block.BlockOperation
			var err error

			var opIndex int
			parsedCongressVotingResultHash := strings.Split(inflationPF.VotingResult, "-") //0:TxHash, 1:Index
			if len(parsedCongressVotingResultHash) != 2 {
				return errors.InvalidOperation
			}
			txHash := parsedCongressVotingResultHash[0]
			if opIndex, err = strconv.Atoi(parsedCongressVotingResultHash[1]); err != nil {
				return errors.InvalidOperation
			}

			if bo, err = block.GetBlockOperationWithIndex(st, txHash, opIndex); err != nil {
				return err
			}

			if bo.Type != operation.TypeCongressVotingResult {
				return errors.InvalidOperation
			}
			var operationBody operation.Body
			if operationBody, err = operation.UnmarshalBodyJSON(bo.Type, bo.Body); err != nil {
				return err
			}

			var o operation.CongressVotingResult
			var ok bool
			if o, ok = operationBody.(operation.CongressVotingResult); !ok {
				return errors.TypeOperationBodyNotMatched
			}
			congressVotingHash = o.CongressVotingHash
		}

		var congressVoting operation.CongressVoting
		{
			var bo block.BlockOperation
			var err error
			var opIndex int
			parsedCongressVotingHash := strings.Split(congressVotingHash, "-") //0:TxHash, 1:Index
			if len(parsedCongressVotingHash) != 2 {
				return errors.InvalidOperation
			}
			txHash := parsedCongressVotingHash[0]
			if opIndex, err = strconv.Atoi(parsedCongressVotingHash[1]); err != nil {
				return errors.InvalidOperation
			}

			if bo, err = block.GetBlockOperationWithIndex(st, txHash, opIndex); err != nil {
				return err
			}

			if bo.Type != operation.TypeCongressVoting {
				return errors.InvalidOperation
			}
			var operationBody operation.Body
			if operationBody, err = operation.UnmarshalBodyJSON(bo.Type, bo.Body); err != nil {
				return err
			}

			var o operation.CongressVoting
			var ok bool
			if o, ok = operationBody.(operation.CongressVoting); !ok {
				return errors.TypeOperationBodyNotMatched
			}
			congressVoting = o
		}

		if congressVoting.Amount != inflationPF.Amount {
			return errors.InflationPFAmountMissMatched
		}

		if congressVoting.FundingAddress != inflationPF.FundingAddress {
			return errors.InflationPFFundingAddressMissMatched
		}

	case operation.TypeCongressVoting:
		//the CongressAddress is owned by blockchainOS. It is temporally check.
		//TODO: When a node of BosNet is operated by anonymous then it will be removed.
		if source.Address != config.CongressAccountAddress {
			return errors.CongressAddressMisMatched
		}
	case operation.TypeCongressVotingResult:
		//the CongressAddress is owned by blockchainOS. It is temporally check.
		//TODO: When a node of BosNet is operated by anonymous then it will be removed.
		if source.Address != config.CongressAccountAddress {
			return errors.CongressAddressMisMatched
		}

		var ok bool
		var cvResult operation.CongressVotingResult
		if cvResult, ok = op.B.(operation.CongressVotingResult); !ok {
			return errors.TypeOperationBodyNotMatched
		}

		var opIndex int
		parsedCongressVotingResultHash := strings.Split(cvResult.CongressVotingHash, "-") //0:TxHash, 1:Index
		if len(parsedCongressVotingResultHash) != 2 {
			return errors.InvalidOperation
		}
		txHash := parsedCongressVotingResultHash[0]
		if opIndex, err = strconv.Atoi(parsedCongressVotingResultHash[1]); err != nil {
			return errors.InvalidOperation
		}

		if _, err = block.GetBlockOperationWithIndex(st, txHash, opIndex); err != nil {
			return err
		}

	default:
		return errors.UnknownOperationType
	}
	return nil
}
