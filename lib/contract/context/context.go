package context

import "boscoin.io/sebak/lib/contract/payload"

type Context interface {
	SenderAddress() string
	PutDeployCode(*payload.DeployCode) error
}
