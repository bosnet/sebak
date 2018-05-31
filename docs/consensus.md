# ISAAC Consensus Protocol

## States

There are 4 state, `INIT` → `SIGN` → `ACCEPT` → `ALL-CONFIRM`


## State Transition

### `INIT` → `SIGN`

In this state, trying to broadcast new incoming transaction from client to the entire network.

* The new incoming transaction(`Txm`) from client will be assigned `INIT` state.

* broadcast the ballot(`Ba`), which generated from`Txm` to the validators of current node
    * Each node receives `Ba` and make their own `Ba` and then, broadcast it

* Each validator will wait `Ba`, to check 100% threshold to receive from all the **connected** validators.
    * If passed,
        - `Ba` will be added to 'voting box' and removed from 'waiting box'


### `SIGN` → `ACCEPT`

In this state, each node tries to validate `Ba` and it's `Txm` and update `Ba` with voting result(we call it 'voting hole'). If `Ba` in `SIGN`, it means that the `Txm` of `Ba` is fully spreaded to the entire network.

* The validation result, 'voting hole' will be set in ballot
* broadcasting `Ba` and * check threshold. Not like `INIT`, the the threshold of `SIGN` should not be 100%. 100% threshold of `INIT` can spread a transaction to entire network.

### `ACCEPT` → `ALL-CONFIRM`

In this state, check all the network is reached to `ACCEPT` and ready to store it.

* broadcast `Ba` and check threshold

### `ALL-CONFIRM`

In this state, confirmed `Ba` and it's `Txm` will be stored in block and the consensus process will be ended.
