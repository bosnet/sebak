# How transactions are shared

All transactions must be shared among nodes in network. This is because all nodes must validate the transactions in the proposed ballot by itself and vote with the result. Also, because of efficiency, only hash of the transaction is included in the ballot.

## Transaction Protocol

Transaction Protocol is used for sharing transactions.

## Process

1. A client sends a transaction request message to the node.
1. The node that received the message proceeds with the transaction in the message.
1. The node checks the format of the transaction.
    * If well-formed, it goes to the next step.
    * If malformed, it discards the transaction and stops this process.
1. If the same transaction is already in the transaction pool, it stops this process.
1. The node [validates](./tech_how_to_check_a_ballot_is_valid.md) the transaction.
    * If valid, it is stored in transaction pool.
    * If invalid, it discards the transaction and stops this process.
1. The node broadcasts the transaction message to all nodes except for the sender.