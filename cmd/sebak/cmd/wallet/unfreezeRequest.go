//
// Implement CLI for payment, account creation, and freezing requests
//
package wallet

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"

	cmdcommon "boscoin.io/sebak/cmd/sebak/common"
	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/common/keypair"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/transaction"
	"boscoin.io/sebak/lib/transaction/operation"
)

var (
	UnfreezeRequestCmd *cobra.Command
)

func init() {
	UnfreezeRequestCmd = &cobra.Command{
		Use:   "unfreezeRequest <sender secret seed>",
		Short: "Request unfreezing for the frozen account",
		Args:  cobra.ExactArgs(1),
		Run: func(c *cobra.Command, args []string) {
			var err error
			var frozenAccountBalance common.Amount
			var sender keypair.KP
			var endpoint *common.Endpoint

			// Sender's secret seed
			if sender, err = keypair.Parse(args[0]); err != nil {
				cmdcommon.PrintFlagsError(c, "<sender secret seed>", err)
			} else if _, ok := sender.(*keypair.Full); !ok {
				cmdcommon.PrintFlagsError(c, "<sender secret seed>", fmt.Errorf("Provided key is an address, not a secret seed"))
			}

			// Check a network ID was provided
			if len(flagNetworkID) == 0 {
				cmdcommon.PrintFlagsError(c, "--network-id", fmt.Errorf("A --network-id needs to be provided"))
			}

			if endpoint, err = common.ParseEndpoint(flagEndpoint); err != nil {
				cmdcommon.PrintFlagsError(c, "--endpoint", err)
			}

			var tx transaction.Transaction
			var connection *common.HTTP2Client
			var senderAccount block.BlockAccount

			// Keep-alive ignores timeout/idle timeout
			if connection, err = common.NewHTTP2Client(0, 0, true); err != nil {
				log.Fatal("Error while creating network client: ", err)
				os.Exit(1)
			}
			client := network.NewHTTP2NetworkClient(endpoint, connection)

			if senderAccount, err = getSenderDetails(client, sender); err != nil {
				log.Fatal("Could not fetch sender account: ", err)
				os.Exit(1)
			}

			if senderAccount.Linked == "" {
				fmt.Printf("Account is not frozen account")
				os.Exit(1)
			}
			if flagVerbose == true {
				fmt.Println("Account before transaction: ", senderAccount)
			}

			// Check that account's balance is enough before unfreezing.
			{
				frozenAccountBalance = senderAccount.GetBalance()
				if frozenAccountBalance == 0 {
					fmt.Println("Already unfreezed account")
					os.Exit(1)
				}
			}

			tx = makeTransactionUnfreezingRequest(sender, senderAccount.SequenceID)

			tx.Sign(sender, []byte(flagNetworkID))

			// Send request
			var retbody []byte
			if flagDry == true || flagVerbose == true {
				fmt.Println(tx)
			}
			if flagDry == false {
				if retbody, err = client.SendTransaction(tx); err != nil {
					log.Fatal("Network error: ", err, " body: ", string(retbody))
					os.Exit(1)
				}
			}
		},
	}
	UnfreezeRequestCmd.Flags().StringVar(&flagEndpoint, "endpoint", flagEndpoint, "endpoint to send the transaction to (https / memory address)")
	UnfreezeRequestCmd.Flags().StringVar(&flagNetworkID, "network-id", flagNetworkID, "network id")
	UnfreezeRequestCmd.Flags().BoolVar(&flagDry, "dry-run", flagDry, "Print the transaction instead of sending it")
	UnfreezeRequestCmd.Flags().BoolVar(&flagVerbose, "verbose", flagVerbose, "Print extra data (transaction sent)")
}

//
/// Make a full transaction, with a single unfreezing request operation in it
///
/// Params:
///   kpSource = Sender's keypair.Full seed/address
///   kpDest   = Receiver's keypair.FromAddress address
///   amount   = Amount to send as initial value
///   seqid    = SequenceID of the last transaction
///
/// Returns:
///  `sebak.Transaction` = The generated `Transaction` to do a unfreezing request
///
func makeTransactionUnfreezingRequest(kpSource keypair.KP, seqid uint64) transaction.Transaction {
	opb := operation.NewUnfreezeRequest()

	op := operation.Operation{
		H: operation.Header{
			Type: operation.TypeUnfreezingRequest,
		},
		B: opb,
	}

	txBody := transaction.Body{
		Source:     kpSource.Address(),
		Fee:        common.FrozenFee,
		SequenceID: seqid,
		Operations: []operation.Operation{op},
	}

	tx := transaction.Transaction{
		H: transaction.Header{
			Version: common.TransactionVersionV1,
			Created: common.NowISO8601(),
			Hash:    txBody.MakeHashString(),
		},
		B: txBody,
	}

	return tx
}
