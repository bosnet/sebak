# How to select the proposer

The proposer is selected based on block height and round.

## Block Height

Last confirmed block height.

## Round

Round is a variable used within one block generation cycle.
When some nodes do not run normally, the round is needed to select a normal node for proposer.

## Select Proposer Function

1. Sorting validators by valiator address alphabetically.
1. `n` = (block height + round number) mod len(validators)
1. The proposer is n'th validator

```go
func CalculateProposer(blockHeight int, roundNumber int) string {
	candidates := sort.StringSlice(nr.connectionManager.RoundCandidates())
	candidates.Sort()

	return candidates[(blockHeight + roundNumber)%len(candidates)]
}
```
