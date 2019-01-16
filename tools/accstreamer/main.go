//
// Account streamer is a simple utility for tests
//
// It subscribe to the account, and wait for an account
// to reach a certain balance.
// Once this balance is reached, it exits with a 0 status code.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"boscoin.io/sebak/lib/client"
	"boscoin.io/sebak/lib/common"
)

type Expectation struct {
	address string
	balance common.Amount
}

// This program expects an uneven number of arguments (>3):
// - the server address (without trailing slash)
// - a pair of address + balance
func main () {
	if len(os.Args) < 4 {
		fmt.Println("ERROR: At least three arguments expected")
		os.Exit(1)
	}

	server := os.Args[1]
	args := os.Args[2:]
	if (len(args) % 2) != 0 {
		fmt.Println("ERROR: Arguments should be <server> <address balance>+")
		os.Exit(1)
	}


	var exps []Expectation
	var addresses []string
	for i := 0; i < len(args); i += 2 {
		addresses = append(addresses, args[i])
		exps = append(exps, Expectation{ address: args[i], balance: common.MustAmountFromString(args[i + 1]) })
	}

	cli := client.MustNewClient(server)
	ctx, cancel := context.WithCancel(context.Background())

	handler := func(tx client.Account) {
		// We log the changes so if something fail, we have an history of what the client saw
		tnow := time.Now()
		fmt.Printf("%02d-%d-%d:%s:%s:%d\n", tnow.Hour(), tnow.Minute(), tnow.Second(),
			tx.Address, tx.Balance, tx.SequenceID)
		current := common.MustAmountFromString(tx.Balance)
		activeWatcher := false

		for idx, exp := range exps {
			// This is an address we care about
			if exp.address == tx.Address {
				if exp.balance != current {
					// Not the right balance yet, bail out
					return
				}
				exps[idx].address = ""
				// If there's still active watcher,
				// there is no point in iterating further
				if activeWatcher == true {
					return
				}
			} else if len(exp.address) != 0 {
				// Can only cancel if all the addresses are set to ""
				activeWatcher = true
			}

		}
		if activeWatcher == false {
			cancel()
		}
	}

	if err := cli.StreamAccount(ctx, handler, addresses...); err != nil {
		fmt.Println("ERROR: ", err)
		os.Exit(1)
	}
}
