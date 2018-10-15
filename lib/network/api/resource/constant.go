package resource

const (
	APIVersionV1 = "/v1"
	APIPrefix    = "/api"

	URLAccounts              = APIPrefix + APIVersionV1 + "/accounts/{id}"
	URLAccountTransactions   = APIPrefix + APIVersionV1 + "/accounts/{id}/transactions"
	URLAccountOperations     = APIPrefix + APIVersionV1 + "/accounts/{id}/operations"
	URLTransactions          = APIPrefix + APIVersionV1 + "/transactions"
	URLTransactionByHash     = APIPrefix + APIVersionV1 + "/transactions/{id}"
	URLTransactionOperations = APIPrefix + APIVersionV1 + "/transactions/{id}/operations"
	URLOperations            = APIPrefix + APIVersionV1 + "/operations/{id}"
)
