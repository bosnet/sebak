package resource

const (
	APIVersionV1 = "/v1"
	APIPrefix    = "/api"

	URLAccounts              = APIPrefix + APIVersionV1 + "/accounts/{id}"
	URLAccountTransactions   = APIPrefix + APIVersionV1 + "/accounts/{id}/transactions"
	URLAccountOperations     = APIPrefix + APIVersionV1 + "/accounts/{id}/operations"
	URLAccountFrozenAccounts = APIPrefix + APIVersionV1 + "/accounts/{id}/frozen-accounts"
	URLFrozenAccounts        = APIPrefix + APIVersionV1 + "/frozen-accounts"
	URLTransactions          = APIPrefix + APIVersionV1 + "/transactions"
	URLTransactionByHash     = APIPrefix + APIVersionV1 + "/transactions/{id}"
	URLTransactionOperations = APIPrefix + APIVersionV1 + "/transactions/{id}/operations"
	URLTransactionHistory    = APIPrefix + APIVersionV1 + "/transactions/{id}/history"
	URLOperations            = APIPrefix + APIVersionV1 + "/operations/{id}"
	URLBlocks                = APIPrefix + APIVersionV1 + "/blocks/{id}"
)
