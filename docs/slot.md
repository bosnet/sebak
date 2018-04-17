## Slot

* The transition of `INIT` -> `SIGN` means that the new transaction is added to 'slot'.

## Room

* The number of room in slot: infinit


### Waiting Slot:

Every transactions will be added to the 'waiting slot'.


### Voting Slot

The `SIGN`ed ballot will be appended to the 'voting slot'. The transactions in this slot will be voted.


### Reserv4ed Slot

If the transaction is expired in the 'voting slot', it will be added to the 'reserved slot'. If new ***Ba*** is received and it is found in 'reserved slot', the consensus will be proceeded.

* If ballot in 'voting slot' is expired in 1 minute, it will be moved to 'reserved slot'
* If ballot in 'reserved slot' is expired in 5 minutes, it will be **removed** completely.
