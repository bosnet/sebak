## Boxes

* The transition of `INIT` -> `SIGN` means that the new transaction is ready to vote.
* if ballot is remaining in up to 2 minute in 'waiting box' and 'voting box' without new incoming ballot, it will be moved to 'reserved box'.
* if ballot is remaining in up to 2 minute in 'reserved box', it will be removed permanantly.

## Box

* The number of room in box: *infinit*


### Waiting Box

Every new incoming transactions will be added to the 'waiting box'.


### Voting Box

The `SIGN`ed ballot will be added to the 'voting box'. The ballots in this box will be voted by consensus process.


### Reserved Box

If the ballot is expired in the 'waiting box' or 'voting box', it will be added to the 'reserved box'. If new ballot is received and it can be found in 'reserved box', the consensus will be proceeded again.
