//
// Implement CLI for payment, account creation, and freezing requests
//
package wallet

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	cmdcommon "boscoin.io/sebak/cmd/sebak/common"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/network"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/transaction"
	"github.com/spf13/cobra"
	"github.com/stellar/go/keypair"
)

var (
	flagNetworkID     string = common.GetENVValue("SEBAK_NETWORK_ID", "")
	PaymentCmd        *cobra.Command
	flagEndpoint      string
	flagCreateAccount bool
	flagDry           bool
	flagFreeze        bool
	flagVerbose       bool
)

func init() {
	PaymentCmd = &cobra.Command{
		Use:   "payment <receiver pubkey> <amount> <sender secret seed>",
		Short: "Send <amount> BOSCoin from one wallet to another",
		Args:  cobra.ExactArgs(3),
		Run: func(c *cobra.Command, args []string) {
			var err error
			var amount common.Amount
			var newBalance common.Amount
			var sender keypair.KP
			var receiver keypair.KP
			var endpoint *common.Endpoint

			// Receiver's public key
			if receiver, err = keypair.Parse(args[0]); err != nil {
				cmdcommon.PrintFlagsError(c, "<receiver public key>", err)
			} else if _, err = receiver.Sign([]byte("witness")); err == nil {
				cmdcommon.PrintFlagsError(c, "<receiver public key>", fmt.Errorf("Provided key is a secret seed, not an address"))
			}

			// Amount
			if amount, err = cmdcommon.ParseAmountFromString(args[1]); err != nil {
				cmdcommon.PrintFlagsError(c, "<amount>", err)
			}
			if flagFreeze == true && (amount%common.Unit) != 0 {
				cmdcommon.PrintFlagsError(c, "<amount>",
					fmt.Errorf("Amount should be an exact multiple of %v when --freeze is provided", common.Unit))
			}

			// Sender's secret seed
			if sender, err = keypair.Parse(args[2]); err != nil {
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

			// TODO: Validate input transaction (does the sender have enough money?)

			// At the moment this is a rather crude implementation: There is no support for pooling of transaction,
			// 1 operation == 1 transaction
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

			if flagVerbose == true {
				fmt.Println("Account before transaction: ", senderAccount)
			}

			// Check that account's balance is enough before sending the transaction
			{
				newBalance, err = senderAccount.GetBalance().Sub(amount)
				if err == nil {
					newBalance, err = newBalance.Sub(common.BaseFee)
				}

				if err != nil {
					fmt.Printf("Attempting to draft %v GON (+ %v fees), but sender account only have %v GON\n",
						amount, common.BaseFee, senderAccount.GetBalance())
					os.Exit(1)
				}
			}

			// TODO: Validate that the account doesn't already exists
			if flagFreeze {
				tx = makeTransactionCreateAccount(sender, receiver, amount, senderAccount.SequenceID, sender.Address())
			} else if flagCreateAccount {
				tx = makeTransactionCreateAccount(sender, receiver, amount, senderAccount.SequenceID, "")
			} else {
				tx = makeTransactionPayment(sender, receiver, amount, senderAccount.SequenceID)
			}

			tx.Sign(sender, []byte(flagNetworkID))

			// Send request
			var retbody []byte
			if flagDry == true || flagVerbose == true {
				fmt.Println(tx)
			}
			if flagDry == false {
				if retbody, err = client.SendMessage(tx); err != nil {
					log.Fatal("Network error: ", err, " body: ", retbody)
					os.Exit(1)
				}
			}
			if flagVerbose == true {
				time.Sleep(5 * time.Second)
				if recv, err := getSenderDetails(client, receiver); err != nil {
					fmt.Println("Account ", receiver.Address(), " did not appear after 5 seconds")
				} else {
					fmt.Println("Receiver account after 5 seconds: ", recv)
				}
			}
		},
	}
	PaymentCmd.Flags().StringVar(&flagEndpoint, "endpoint", flagEndpoint, "endpoint to send the transaction to (https / memory address)")
	PaymentCmd.Flags().StringVar(&flagNetworkID, "network-id", flagNetworkID, "network id")
	PaymentCmd.Flags().BoolVar(&flagCreateAccount, "create", flagCreateAccount, "Whether or not the account should be created")
	PaymentCmd.Flags().BoolVar(&flagFreeze, "freeze", flagFreeze, "When present, the payment is a frozen account creation. Imply --create.")
	PaymentCmd.Flags().BoolVar(&flagDry, "dry-run", flagDry, "Print the transaction instead of sending it")
	PaymentCmd.Flags().BoolVar(&flagVerbose, "verbose", flagVerbose, "Print extra data (transaction sent, before/after balance...)")
}

///
/// Make a full transaction, with a single operation to create an account
///
/// TODO:
///   Move to lib (it was 'borrowed' from test code)
///
/// Params:
///   kpSource = Sender's keypair.Full seed/address
///   kpDest   = Newly created account's address
///   amount   = Amount to send as initial value
///   seqid    = SequenceID of the last transaction
///   target   = Address of the linked account, if we're creating a frozen account
///
/// Returns:
///   `sebak.Transaction` = The generated `Transaction` creating the account
///
func makeTransactionCreateAccount(kpSource keypair.KP, kpDest keypair.KP, amount common.Amount, seqid uint64, target string) transaction.Transaction {
	opb := transaction.NewOperationBodyCreateAccount(kpDest.Address(), amount, target)

	op := transaction.Operation{
		H: transaction.OperationHeader{
			Type: transaction.OperationCreateAccount,
		},
		B: opb,
	}

	txBody := transaction.TransactionBody{
		Source:     kpSource.Address(),
		Fee:        common.BaseFee,
		SequenceID: seqid,
		Operations: []transaction.Operation{op},
	}

	tx := transaction.Transaction{
		T: "transaction",
		H: transaction.TransactionHeader{
			Created: common.NowISO8601(),
			Hash:    txBody.MakeHashString(),
		},
		B: txBody,
	}

	return tx
}

///
/// Make a full transaction, with a single payment operation in it
///
/// TODO:
///   Move to lib (it was 'borrowed' from test code)
///
/// Params:
///   kpSource = Sender's keypair.Full seed/address
///   kpDest   = Receiver's keypair.FromAddress address
///   amount   = Amount to send as initial value
///   seqid    = SequenceID of the last transaction
///
/// Returns:
///  `sebak.Transaction` = The generated `Transaction` to do a payment
///
func makeTransactionPayment(kpSource keypair.KP, kpDest keypair.KP, amount common.Amount, seqid uint64) transaction.Transaction {
	opb := transaction.NewOperationBodyPayment(kpDest.Address(), amount)

	op := transaction.Operation{
		H: transaction.OperationHeader{
			Type: transaction.OperationPayment,
		},
		B: opb,
	}

	txBody := transaction.TransactionBody{
		Source:     kpSource.Address(),
		Fee:        common.Amount(common.BaseFee),
		SequenceID: seqid,
		Operations: []transaction.Operation{op},
	}

	tx := transaction.Transaction{
		T: "transaction",
		H: transaction.TransactionHeader{
			Created: common.NowISO8601(),
			Hash:    txBody.MakeHashString(),
		},
		B: txBody,
	}

	return tx
}

///
/// Get the BlockAccount of the sender
///
/// TODO:
///   Move to lib
///
/// Params:
///   conn = Network connection to the node to request
///   sender = sender from which to request the account (only `Address` is used)
///
/// Returns:
///   sebak.BlockAccount = The deserialized block account, or a default-initialized one if an error occured
///   error = `nil` or the error that occured (either network or deserialization)
///
func getSenderDetails(conn *network.HTTP2NetworkClient, sender keypair.KP) (block.BlockAccount, error) {
	var ba block.BlockAccount
	var err error
	var retBody []byte

	//response, err = c.client.Post(u.String(), body, headers)
	if retBody, err = conn.Get("/api/v1/accounts/" + sender.Address()); err != nil {
		return ba, err
	}

	err = json.Unmarshal(retBody, &ba)
	return ba, err
}
