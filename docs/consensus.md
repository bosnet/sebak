# ISAAC Consensus Protocol

## States

There are 4 state, `INIT` → `SIGN` → `ACCEPT` → `ALL-CONFIRM`

* if ***Txm*** and ***Ba*** is expired in 2 minute in 'waiting box'(or 'waiting ballot box') and 'voting box', it will be moved to 'reserved box'


## State Transition

### `INIT` → `SIGN`

In this state, broadcast ***Txm*** to the entire network.

* The new incoming transaction('Txm') from client will be assigned `INIT` state.

* validate
    * key points
        * ***Txm*** has valid format and can be unserializable
        * is already in block?
        * is already in 'waiting box', 'voting box' or 'reserved box'

    * if validated,
        * store ***Txm*** in ***Txh***
    * if not,
        * abondon it

* if validated
    1. check in the `voting box` or `reserved box`
    	- if exists in the `voting box`,
    		1. ignore it
    	- if exists in the `reserved box`,
    		- if state is `init`
    			1. the ballot in the `reserved box` moves to `waiting box`
    		- if not,
    			1. ignore it
    1. check in the `waiting box`
    	- if not,
    		1. add it and node also add with current node
    	- if exists,
    		1. add to `waiting box`


* broadcast ***Txm*** to the validators of node
    * Each node receive ***Txm*** and make ***Ba*** and then, broadcast it

* The node will wait ***Ba***, to check 100% threshold to receive from all the **connected** validators
    * If passed,
        - ***hash** of ***Txm*** will be added to 'voting box'
        - remove from 'waiting box'
    * If expired in 2 minute, it will be moved to 'reserved box'

* when adding 'voting box', the new ***Ba*** will be checked,
    - is already in block?
    - is already in 'voting box', 'reserved box' or 'reserved box'


### `SIGN` → `ACCEPT`

In this state, validate ***Txm*** and update ***Ba*** with voting result. If ***Ba*** in ```SIGN`, it means that the ***Txm*** of ***Ba*** is fully spreaded to the entire network.

* validate ***Txm***
    * key points
        * ***hash*** is valid
            * exists in block
            * compare with the hashed of ***data**
        * ***signature*** is valid
            * valid ***signature*** from ***hash***
        * ***created*** time is within 5 seconds before and after
        * ***sender*** exists
        * ***operations*** is not empty
        * total ***amount*** + (***fee*** * number of ***operation***) is lower than the latest balance

* validate ***Opm***
    * key points
        * ***type*** is valid
            * `create-account`
            * `payment`
        * ***hash*** is valid
            * compare with the hashed of ***data**
        * ***receiver*** exists
        * ***checkpoint*** points the latest ***checkpoint*** of ***Opb***
        * ***amount*** is lower than latest balance
            * ***amount*** is greater than 0
            * if ***type*** is `create-account`, ***amount*** is greater than ***minimum balance***

* The validation result, ***vote*** will be set in ballot

* broadcast ***Ba***
* check threshold
    * if passed, set to `ACCEPT`
    * if not,
        * stop consensus
        * update ***reason*** in ***Txh***


### `ACCEPT` → `ALL-CONFIRM`

In this state, check all the network is reached to `ACCEPT` and ready to store it

* broadcast ***Ba***
* check threshold
    * if passed, set to `ALL-CONFIRM`
    * if not,
        * stop consensus
        * update ***reason*** in ***Txh***

### `ALL-CONFIRM`

In this state, confirmed ***Txm*** will be stored in ***Txb*** and the consensus process will be ended.

* store ***Txm*** in ***Txb***
* remove from box
