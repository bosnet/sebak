package wallet

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"boscoin.io/sebak/cmd/sebak/common"
	"boscoin.io/sebak/lib"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/network"

	"github.com/spf13/cobra"
	"github.com/stellar/go/keypair"
)

var (
	flagNetworkID     string = sebakcommon.GetENVValue("SEBAK_NETWORK_ID", "")
	PaymentCmd        *cobra.Command
	flagEndpoint      string
	flagCreateAccount bool
)

func init() {
	PaymentCmd = &cobra.Command{
		Use:   "payment <receiver pubkey> <amount> <sender secret seed>",
		Short: "Send <amount> BOSCoin from one wallet to another",
		Args:  cobra.ExactArgs(3),
		Run: func(c *cobra.Command, args []string) {
			var err error
			var amount sebak.Amount
			var newBalance sebak.Amount
			var sender keypair.KP
			var receiver keypair.KP
			var endpoint *sebakcommon.Endpoint

			// Receiver's public key
			if receiver, err = keypair.Parse(args[0]); err != nil {
				common.PrintFlagsError(c, "<receiver public key>", err)
			} else if _, err = receiver.Sign([]byte("witness")); err == nil {
				common.PrintFlagsError(c, "<receiver public key>", fmt.Errorf("Provided key is a secret seed, not an address"))
			}

			// Amount
			if amount, err = common.ParseAmountFromString(args[1]); err != nil {
				common.PrintFlagsError(c, "<amount>", err)
			}

			// Sender's secret seed
			if sender, err = keypair.Parse(args[2]); err != nil {
				common.PrintFlagsError(c, "<sender secret seed>", err)
			} else if _, ok := sender.(*keypair.Full); !ok {
				common.PrintFlagsError(c, "<sender secret seed>", fmt.Errorf("Provided key is an address, not a secret seed"))
			}

			// Check a network ID was provided
			if len(flagNetworkID) == 0 {
				common.PrintFlagsError(c, "--network-id", fmt.Errorf("A --network-id needs to be provided"))
			}

			if endpoint, err = sebakcommon.ParseEndpoint(flagEndpoint); err != nil {
				common.PrintFlagsError(c, "--endpoint", err)
			}

			// TODO: Validate input transaction (does the sender have enough money?)

			// At the moment this is a rather crude implementation: There is no support for pooling of transaction,
			// 1 operation == 1 transaction
			var tx sebak.Transaction
			var connection *sebakcommon.HTTP2Client
			var senderAccount sebak.BlockAccount

			// Keep-alive ignores timeout/idle timeout
			if connection, err = sebakcommon.NewHTTP2Client(0, 0, true); err != nil {
				log.Fatal("Error while creating network client: ", err)
				os.Exit(1)
			}
			client := sebaknetwork.NewHTTP2NetworkClient(endpoint, connection)

			if senderAccount, err = getSenderDetails(client, sender); err != nil {
				log.Fatal("Could not fetch sender account: ", err)
				os.Exit(1)
			}

			// Check that account's balance is enough before sending the transaction
			{
				newBalance, err = sebak.MustAmountFromString(senderAccount.Balance).Sub(amount)
				if err == nil {
					newBalance, err = newBalance.Sub(sebak.BaseFee)
				}

				if err != nil {
					fmt.Printf("Attempting to draft %v GON (+ %v fees), but sender account only have %v GON\n",
						amount, sebak.BaseFee, senderAccount.Balance)
					os.Exit(1)
				}
			}

			// TODO: Validate that the account doesn't already exists
			if flagCreateAccount {
				tx = makeTransactionCreateAccount(sender, receiver, amount, senderAccount.Checkpoint)
			} else {
				tx = makeTransactionPayment(sender, receiver, amount, senderAccount.Checkpoint)
			}

			tx.Sign(sender, []byte(flagNetworkID))

			// Send request
			var retbody []byte
			if retbody, err = client.SendMessage(tx); err != nil {
				log.Fatal("Network error: ", err, " body: ", retbody)
				os.Exit(1)
			}
		},
	}
	PaymentCmd.Flags().StringVar(&flagEndpoint, "endpoint", flagEndpoint, "endpoint to send the transaction to (https / memory address)")
	PaymentCmd.Flags().StringVar(&flagNetworkID, "network-id", flagNetworkID, "network id")
	PaymentCmd.Flags().BoolVar(&flagCreateAccount, "create", flagCreateAccount, "Whether or not the account should be created")
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
///   chkp     = Checkpoint of the last transaction
///
/// Returns:
///   `sebak.Transaction` = The generated `Transaction` creating the account
///
func makeTransactionCreateAccount(kpSource keypair.KP, kpDest keypair.KP, amount sebak.Amount, chkp string) sebak.Transaction {
	opb := sebak.NewOperationBodyCreateAccount(kpDest.Address(), amount)

	op := sebak.Operation{
		H: sebak.OperationHeader{
			Type: sebak.OperationCreateAccount,
		},
		B: opb,
	}

	txBody := sebak.TransactionBody{
		Source:     kpSource.Address(),
		Fee:        sebak.BaseFee,
		Checkpoint: chkp,
		Operations: []sebak.Operation{op},
	}

	tx := sebak.Transaction{
		T: "transaction",
		H: sebak.TransactionHeader{
			Created: sebakcommon.NowISO8601(),
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
///   chkp     = Checkpoint of the last transaction
///
/// Returns:
///  `sebak.Transaction` = The generated `Transaction` to do a payment
///
func makeTransactionPayment(kpSource keypair.KP, kpDest keypair.KP, amount sebak.Amount, chkp string) sebak.Transaction {
	opb := sebak.NewOperationBodyPayment(kpDest.Address(), amount)

	op := sebak.Operation{
		H: sebak.OperationHeader{
			Type: sebak.OperationPayment,
		},
		B: opb,
	}

	txBody := sebak.TransactionBody{
		Source:     kpSource.Address(),
		Fee:        sebak.Amount(sebak.BaseFee),
		Checkpoint: chkp,
		Operations: []sebak.Operation{op},
	}

	tx := sebak.Transaction{
		T: "transaction",
		H: sebak.TransactionHeader{
			Created: sebakcommon.NowISO8601(),
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
func getSenderDetails(conn *sebaknetwork.HTTP2NetworkClient, sender keypair.KP) (sebak.BlockAccount, error) {
	var ba sebak.BlockAccount
	var err error
	var retBody []byte

	//response, err = c.client.Post(u.String(), body, headers)
	if retBody, err = conn.Get("/api/account/" + sender.Address()); err != nil {
		return ba, err
	}

	err = json.Unmarshal(retBody, &ba)
	return ba, err
}
