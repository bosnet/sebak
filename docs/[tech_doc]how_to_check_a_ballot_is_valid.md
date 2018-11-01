# How to check a ballot is valid

A node must validate the ballot with transactions from proposer(Please refer to [How the consensus protocol works](./[tech_doc]how_the_consensus_protocol_works.md)). This is because only valid ballot must be agreed and confirmed.

## Ballot Structure

When a ballot is made by the proposer, it has `valid transactions`.

### Received Ballot Validation

A ballot is valid if all transactions are passed the validation check by itself.

### Transaction Validation

A transaction is valid if,
1. it has valid checkpoint,
1. the source account has enough balance to pay for the transaction and
1. it's `Operation`s are valid