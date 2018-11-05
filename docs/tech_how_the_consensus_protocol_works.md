# How the consensus protocol works

# Consensus Protocol

There are two protocols in `SEBAK`. One is `ISAAC` protocol for consensus and another is `Transaction Protocol` for sharing transactions.

## `ISAAC` Protocol
The `ISAAC` is a consensus protocol based on `PBFT`.

## `ISAAC` States
1. `INIT` - Proposed or received a ballot with transactions.

1. `SIGN` - Checked that the ballot is valid by itself.

1. `ACCEPT` - Checked that the ballot is valid by validators.

1. `ALL_CONFIRM` - Confirmed block with the transactions in the ballot.

## Voting Hole

1. `YES` - Agreed

1. `NO` - Disagreed

1. `EXP` - Expired

## Terms

* TIMEOUT_INIT - The timeout for `INIT` state. The default value is 2 sec.

* TIMEOUT_SIGN - The timeout for `SIGN` state. The default value is 2 sec.

* TIMEOUT_ACCEPT - The timeout for `ACCEPT` state. The default value is 2 sec.

* TIMEOUT_ALLCONFIRM - The timeout for `ALLCONFIRM`. This value is not configurable. It is calculated based on block time and proposed time. Please check [How-to-calculate-timeout-allconfirm](./tech_how_to_calculate_timeout_allconfirm.md) in detail.

* B(ISAAC State, Voting Hole) - A ballot with ISAAC state and voting hole.

### Examples

* B(`INIT`, `YES`) - A ballot with ISAAC state `INIT` and voting hole `YES`.

* B(`SIGN`, `NO`) - A ballot with ISAAC state `INIT` and voting hole `NO`.

* B(`ACCEPT`, `EXP`) - A ballot with ISAAC state `INIT` and voting hole `EXP`.

## Voting Process

### Network Start
* At the beginning of the network, the genesis block is saved with block height 1 and round 0.
* The node start with `INIT` state, height 2 and round 0.

### `INIT`
1. The timer is set to TIMEOUT_INIT.
1. The steps to propose transactions are as follows.
   * If the node is [proposer](./tech_how_to_select_the_proposer.md) of this round broadcasts a B(`INIT`, `YES`),
      * The ballot includes valid transactions(only hashes) or empty transaction.
      * When it broadcasts the ballot, the node goes to the next state.
   * If the node is not proposer, it waits for and receive the B(`INIT`, `YES`).
      * When it receives the ballot, the node goes to the next state.
   * When the timer expires, the node goes to the next state.

### `SIGN`
1. The timer is reset to TIMEOUT_SIGN
1. The node checks [the proposed ballot is valid](./tech_how_to_check_a_ballot_is_valid.md).
   * If the proposed ballot is valid, the node broadcasts B(`SIGN`, `YES`).
   * If the proposed ballot is invalid, the node broadcasts B(`SIGN`, `NO`).
   * If the proposed ballot is empty, the node broadcasts B(`SIGN`, `EXP`).
1. Each node receives ballots and when,
   * the number of B(`SIGN`, `YES`) is greater than or equal to 2/3 of validators, the node broadcasts B(`ACCEPT`, `YES`).
   * the number of B(`SIGN`, `NO`) is greater than or equal to 2/3 of validators, the node broadcasts B(`ACCEPT`, `NO`).
   * it is not possible to satisfy the above two conditions, the node broadcasts B(`ACCEPT`, `EXP`). The specific case is,
      * even if the remaining votes are all B(`SIGN`, `YES`), when the number of B(`SIGN`, `YES`) does not exceed 2/3 of validators.
      * even if the remaining votes are all B(`SIGN`, `NO`), when the number of B(`SIGN`, `NO`) does not exceed 2/3 of validators.
   * When the timer expires, the node broadcasts B(`ACCEPT`, `EXP`).
1. The node goes to the next state.

### `ACCEPT`
1. The timer is reset to TIMEOUT_ACCEPT
1. Each node receives ballots and,
   * if the number of B(`ACCEPT`, `YES`) is greater than or equal to 2/3 of validators, the node goes to the next state.
   * if it is not possible to satisfy the above condition, the node goes back to `INIT` state with same height and round + 1. The specific case is,
      * even if the remaining votes are all B(`ACCEPT`, `YES`), when the number of B(`ACCEPT`, `YES`) does not exceed 2/3 of validators.
      * even if the remaining votes are all B(`ACCEPT`, `NO`), when the number of B(`ACCEPT`, `NO`) does not exceed 2/3 of validators.
   * when the timer expires, the node goes back to `INIT` state with same height and round + 1.

### `ALL-CONFIRM`
1. the node confirms and saves the block with proposed transactions even though it is empty.
1. the node filters [invalid transactions](./tech_how_to_check_a_ballot_is_valid.md#transaction-validation) in the transaction pool.
1. The node waits for [TIMEOUT_ALLCONFIRM](./tech_how_to_calculate_timeout_allconfirm.md).
1. It goes to `INIT` state with height + 1 and round 0.

## Transaction Protocol
1. [`Transaction Protocol`](./tech_how_transactions_are_shared.md) is a protocol for sharing transactions between the nodes.