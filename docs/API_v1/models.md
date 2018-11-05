## Data Structures

### Account
+ _links 
    + operations
        + href: `/api/v1/accounts/GDEPYGGALPJ5HENXCNOQJPPDOQMA2YAXPERZ4XEAKVFFJJEVP4ZBK6QI/operations{?cursor,limit,order}`
        + templated: true (boolean)
    + self
        + href: `/api/v1/accounts/GDEPYGGALPJ5HENXCNOQJPPDOQMA2YAXPERZ4XEAKVFFJJEVP4ZBK6QI` (string)
    + transactions
        + href: `/api/v1/accounts/GDEPYGGALPJ5HENXCNOQJPPDOQMA2YAXPERZ4XEAKVFFJJEVP4ZBK6QI/transactions{?cursor,limit,order}` 
        + templated: true (boolean)
+ address: GDEPYGGALPJ5HENXCNOQJPPDOQMA2YAXPERZ4XEAKVFFJJEVP4ZBK6QI (string, required) - The account’s public key encoded into a base32 string representation.
+ balance: 10000000000000000000 (string,required) - GON. 1 BOS = 10,000,000 GON
+ sequence_id: 0 (number,required) - The Current sequence number. It needed to submitting a transaction from this account

### Transactions
+ _embedded
    + records
        + _links
            + account
                + href: `/api/v1/accounts/GDIRF4UWPACXPPI4GW7CMTACTCNDIKJEHZK44RITZB4TD3YUM6CCVNGJ`
            + operations
                + href: `/api/v1/transactions/7nLuyg8radTExzBM2WhG37AwohBwEySBw4vj2xdtdjAs/operations{?cursor,limit,order}`
                + templated: true
            + self
                + href: `/api/v1/transactions`
        + created: 2018-11-02T14:09:33.019606000+09:00
        + fee: `10000`
        + hash: 7nLuyg8radTExzBM2WhG37AwohBwEySBw4vj2xdtdjAs
        + operation_count: 1
        + sequence_id: 0
        + source: GDIRF4UWPACXPPI4GW7CMTACTCNDIKJEHZK44RITZB4TD3YUM6CCVNGJ
+ _links
    + next
        + href: /api/v1/account/GDEPYGGALPJ5HENXCNOQJPPDOQMA2YAXPERZ4XEAKVFFJJEVP4ZBK6QI/transactions?limit=100&reverse=false
    + prev
        + href: /api/v1/account/GDEPYGGALPJ5HENXCNOQJPPDOQMA2YAXPERZ4XEAKVFFJJEVP4ZBK6QI/transactions?limit=100&reverse=true
    + self
        + href: /api/v1/account/GDEPYGGALPJ5HENXCNOQJPPDOQMA2YAXPERZ4XEAKVFFJJEVP4ZBK6QI/transactions
    
### Operations
+ _embedded
    + records: null
+ _links
    + next
        + href: `/api/v1/accounts/GDEPYGGALPJ5HENXCNOQJPPDOQMA2YAXPERZ4XEAKVFFJJEVP4ZBK6QI/operations?limit=100&reverse=false`
    + prev
        + href: `/api/v1/accounts/GDEPYGGALPJ5HENXCNOQJPPDOQMA2YAXPERZ4XEAKVFFJJEVP4ZBK6QI/operations?limit=100&reverse=true`
    + self
        + href: `/api/v1/accounts/GDEPYGGALPJ5HENXCNOQJPPDOQMA2YAXPERZ4XEAKVFFJJEVP4ZBK6QI/operations`

### Transaction Payment
+ H 
    + version: `` - Transaction version
    + created: `2018-01-01T00:00:00.000000000Z` - Created time of the transaction.
    + signature: `5iuxEWTtrXnBDiFXvW49jPA6crfKWFes9nXx6scHXBRysvvEMWdTTE8A5yBoqgFMuvpoLMt9ycF8FA9jz1Qyug5k` - Signature signed by source account
+ B
    + source: GDIRF4UWPACXPPI4GW7CMTACTCNDIKJEHZK44RITZB4TD3YUM6CCVNGJ - Source account
    + fee: 10000 - The fee paid by the source account for this transaction. Minimum is 10000 GON
    + sequence_id: 1 - The last sequence number of the source account
    + operations (array):
        + (object):
            + H 
                + type: "payment" - operation type. ex. payment, create-account
            + B
                + target: GDEPYGGALPJ5HENXCNOQJPPDOQMA2YAXPERZ4XEAKVFFJJEVP4ZBK6QI - The funded account's public key
                + amount: 1000000000000- amount in GON

### Transaction Post
+ _embedded
    + records (array)
        + (object):
            + _links
                + account 
                    + href: `/api/v1/accounts/GD6AU7AB4X4Z6SL3JGPD7QMDRJX77USEXTBHDVOQ73IGUKM27LUVBVYH`
                + operations 
                    + href: `/api/v1/transactions/CdigMiMHTaSaFjg9ZMH6hsWXzAee8xQokYWUS1NEL1GC/operations{?cursor,limit,order}`
                    + templated : true
                + self
                    + href : `/api/v1/transactions`
        created: `2018-04-17T5:07:31.000000000Z`
        fee: `10000` - The fee paid by the source account for this transaction. Minimum is 10000 GON
        hash: CdigMiMHTaSaFjg9ZMH6hsWXzAee8xQokYWUS1NEL1GC (string,required) - Hash of transaction
        operation_count: 1
        sequence_id: 0 - The last sequence number of the source account
        source: GD6AU7AB4X4Z6SL3JGPD7QMDRJX77USEXTBHDVOQ73IGUKM27LUVBVYH
+ _links
    + next
        + href: /transactions?cursor=%122018-11-02T14%3A13%3A39.238098000%2B09%3A00-0edd4886-de5e-11e8-befa-8c85901cfacd&limit=100&reverse=false
    + prev:
        + href: /transactions?cursor=%122018-11-02T14%3A13%3A39.238098000%2B09%3A00-0edd4886-de5e-11e8-befa-8c85901cfacd&limit=100&reverse=true
    + self:
        + href: /api/v1/transactions

### Transaction
+ _links 
    + account
        + href: `/api/v1/accounts/GDIRF4UWPACXPPI4GW7CMTACTCNDIKJEHZK44RITZB4TD3YUM6CCVNGJ`
    + operations
        + href: `/api/v1/transactions/ghf6msRhE4jRf5DPib9UHD1msadvmZs9o53V9FQTb11/operations{?cursor,limit,order}`
        + templated: true
    + self
        + href: `/api/v1/transactions/`
+ created: `2018-09-12T09:08:35.157472400Z` - Created time of the transaction. It is set by wallet
+ fee: `10000` (string,required) - The fee paid by the source account
+ hash: `ghf6msRhE4jRf5DPib9UHD1msadvmZs9o53V9FQTb11` (string,required) - Hash of transaction. //TODO: link for the details
+ operation_count: 1 (number) - The number of operations in this transaction.
+ sequence_id: 0 (number) - the Sequence number of the source account.
+ source: `GDIRF4UWPACXPPI4GW7CMTACTCNDIKJEHZK44RITZB4TD3YUM6CCVNGJ` (string,required) -

### TransactionHistory
+ _links
    + account
        + href: `/api/v1/accounts/GDIRF4UWPACXPPI4GW7CMTACTCNDIKJEHZK44RITZB4TD3YUM6CCVNGJ`
    + self
        + href: `/api/v1/transactions`
    + transaction
        + href: `/api/v1/transactions/7nLuyg8radTExzBM2WhG37AwohBwEySBw4vj2xdtdjAs`
    + confirmed: `2018-11-02T14:09:33.021645000+09:00` - Modified time of the transaction history.
    + created: `2018-11-02T14:09:33.019606000+09:00` - Created time of the transaction. It is set by wallet
    + hash: `7nLuyg8radTExzBM2WhG37AwohBwEySBw4vj2xdtdjAs` (string,required) - Hash of transaction. //TODO: link for the details
    + source: `GDIRF4UWPACXPPI4GW7CMTACTCNDIKJEHZK44RITZB4TD3YUM6CCVNGJ` (string,required) - source account
    + status: `confirmed` (string,required) - three categories of status; submitted, confirmed, rejected
        
### Operation
+ _embedded
    + records:(array)
        + (object):
            + _links:
                + self
                    + href: `/api/v1/operations/F6SEv2QhgwZwxUARbRacxyZaufzcTxdYDXJBpvf7pNAj-7nLuyg8radTExzBM2WhG37AwohBwEySBw4vj2xdtdjAs`
                + transaction
                    + href: `/api/v1/transactions/7nLuyg8radTExzBM2WhG37AwohBwEySBw4vj2xdtdjAs` 
        + body
            + target: GDEPYGGALPJ5HENXCNOQJPPDOQMA2YAXPERZ4XEAKVFFJJEVP4ZBK6QI - The funded account's public key
            + amount: `1000000000000` - amount in GON
        + hash: F6SEv2QhgwZwxUARbRacxyZaufzcTxdYDXJBpvf7pNAj-7nLuyg8radTExzBM2WhG37AwohBwEySBw4vj2xdtdjAs - Hash of operation
        + source: GDIRF4UWPACXPPI4GW7CMTACTCNDIKJEHZK44RITZB4TD3YUM6CCVNGJ - Source account
        + tx_hash: 7nLuyg8radTExzBM2WhG37AwohBwEySBw4vj2xdtdjAs - Hash of transaction
        + type: create-account  - operation type. ex. payment, create-account
    + _links
        + next
            + href: `/api/v1/transactions/7nLuyg8radTExzBM2WhG37AwohBwEySBw4vj2xdtdjAs/operations?limit=100&reverse=false`
        + prev
            + href: `/api/v1/transactions/7nLuyg8radTExzBM2WhG37AwohBwEySBw4vj2xdtdjAs/operations?limit=100&reverse=true`
        + self
            + href: `/api/v1/transactions/7nLuyg8radTExzBM2WhG37AwohBwEySBw4vj2xdtdjAs/operations`

### Problem
+ status:  500 (number)
+ title: `problem error message`
+ type: `https://boscoin.io/sebak/error/{error_code}`


### Problem NotFound
+ status: 400 (number)
+ title: `does not exists` 
+ type: `https://boscoin.io/sebak/error/128`
