# How to calculate timeout allconfirm

## Background
The `SEBAK` system should have an average block confirmation time of 5 seconds (configurable). Because the BOScoin blockchain has to issue a certain amount of coins periodically according to the white paper, such as the Bitcoin blockchain, and the criterion is block height. Therefore, if the average time for confirming a block is not constant, then the coin issuance is not constant so various problems arise.

## Solution
To make the average block time be constant, the node sets an interval of TIMEOUT_ALLCONFIRM after consensus and then the block is confirmed and saved.

### TIMEOUT_ALLCONFIRM
There are two purpose of TIMEOUT_ALLCONFIRM. One is to make the block time be constant as described above, and the other is to put a interval to gather transactions before suggesting the next block.
I will briefly explain how TIMEOUT_ALLCONFIRM is calculated, for example.
Assume that the system needs to have a block confirm time of 5 seconds. If the average block generation time is longer than 5 seconds, the node should decrease this block generation time by making this variable as small as possible. If it is shorter than 5 seconds, the node should increase it.

### In detail
If you do not want to know the details, you can skip the following information.
```
func CalculateTimeoutAllConfirm(goal, average, untilNow, delta) Duration {
  if average >= goal {
    result = goal - delta - untilNow
  } else {
    result = goal + delta - untilNow
  }
  if result < 0 {
    result = 0
  }
  return result
}
```
* goal: the goal of block generation time.
* average: the average of block generation time until now.
* untilNow: the time from proposed to agreement of this block.
* delta: the variable to prevent changes sharply of block creation time.

For example,
* goal: 5 sec, average: 7 sec, untilNow: 3 sec, delta: 1 sec
   1. TIMEOUT_ALLCONFIRM = 5 - 1 - 3 = 1.
   1. In other words, since the average(7) is longer than the goal(5), the node should reduce this block time to 4 
   seconds(untilNow(3) + TIMEOUT_ALLCONFIRM(1)).
* goal: 5 sec, average: 3 sec, untilNow: 4 sec, delta: 1 sec
   1. TIMEOUT_ALLCONFIRM = 5 + 1 - 4 = 2.
   1. In other words, since the average(3) is shorter than the goal(5), the node should increase this block time to 6 seconds(untilNow(4) + TIMEOUT_ALLCONFIRM(2)).
* goal: 5 sec, average: 3 sec, untilNow: 7 sec, delta: 1 sec
   1. TIMEOUT_ALLCONFIRM = 5 + 1 - 7 = -1 -> 0.
   1. In other words, since the average(3) is shorter than the goal(5), it is necessary to increase the time taken for consensus. But untilNow(7) has already exceeded the block generation time allowance range(6 = goal + delta), the TIMEOUT_ALLCONFIRM becomes 0.

