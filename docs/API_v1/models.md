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
+ balance: 500000000000 (string) - GON. 1 BOS = 10,000,000 GON
+ linked: "" - linked with freezing account. 
+ sequence_id: 0 (number) - The Current sequence number. It needed to submitting a transaction from this account


### Transactions
+ _embedded
    + records (array)
        + (object):
            + _links
                + account
                    + href: `/api/v1/accounts/GDIRF4UWPACXPPI4GW7CMTACTCNDIKJEHZK44RITZB4TD3YUM6CCVNGJ`
                + operations
                    + href: `/api/v1/transactions/7nLuyg8radTExzBM2WhG37AwohBwEySBw4vj2xdtdjAs/operations{?cursor,limit,order}`
                    + templated: true
                + self
                    + href: `/api/v1/transactions`
        + block: ``
        + created: `2018-11-02T14:09:33.019606000+09:00`
        + fee: 10000
        + hash: `7nLuyg8radTExzBM2WhG37AwohBwEySBw4vj2xdtdjAs`
        + operation_count: 1
        + sequence_id: 0
        + source: `GDIRF4UWPACXPPI4GW7CMTACTCNDIKJEHZK44RITZB4TD3YUM6CCVNGJ`
+ _links
    + next
        + href: `/api/v1/account/GDEPYGGALPJ5HENXCNOQJPPDOQMA2YAXPERZ4XEAKVFFJJEVP4ZBK6QI/transactions?limit=100&reverse=false`
    + prev
        + href: `/api/v1/account/GDEPYGGALPJ5HENXCNOQJPPDOQMA2YAXPERZ4XEAKVFFJJEVP4ZBK6QI/transactions?limit=100&reverse=true`
    + self
        + href: `/api/v1/account/GDEPYGGALPJ5HENXCNOQJPPDOQMA2YAXPERZ4XEAKVFFJJEVP4ZBK6QI/transactions`
    
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
+ T: transaction
+ H 
    + version: `` - Transaction version
    + created: `2018-01-01T00:00:00.000000000Z` - Created time of the transaction.
    + signature: `4ty1Pv7Phc3CEeGLCP8mjZfEC259VR1MBgyVHzQXTcWjuSiwxVQ2AQKxy2HjGTCDrmdE29z8ZNZ6GxuDyEay2p9M` - Signature signed by source account
+ B
    + source: GDIRF4UWPACXPPI4GW7CMTACTCNDIKJEHZK44RITZB4TD3YUM6CCVNGJ - Source account
    + fee: 10000 - The fee paid by the source account for this transaction. Minimum is 10000 GON
    + sequence_id: 0 - The last sequence number of the source account
    + operations (array)
        + (object):
            + H 
                + type: "payment" - operation type. ex. payment, create-account
            + B
                + target: GDTEPFWEITKFHSUO44NQABY2XHRBBH2UBVGJ2ZJPDREIOL2F6RAEBJE4 - The funded account's public key
                + amount: 1000000000000- amount in GON

### Transaction Post 
+ _links  
    + history
        + href: `/api/v1/transactions/7mRUj4cnUPaTrpByojPsT3xoRRdwG6Q9z2eLyCMapQm6/history`
    + self
        + href: `/api/v1/transactions`
+ hash: `7mRUj4cnUPaTrpByojPsT3xoRRdwG6Q9z2eLyCMapQm6`- Hash of transaction.
+ message
    + source: GDIRF4UWPACXPPI4GW7CMTACTCNDIKJEHZK44RITZB4TD3YUM6CCVNGJ - Source account
    + fee: `10000` - The fee paid by the source account for this transaction. Minimum is 10000 GON
    + sequence_id: 0 - The last sequence number of the source account
    + operations (array)
        + (object):
            + H
                + type:create-account - operation type. ex. payment, create-account
            + B
                + target: GDTEPFWEITKFHSUO44NQABY2XHRBBH2UBVGJ2ZJPDREIOL2F6RAEBJE4 - The funded account's public key
                + amount: 1000000000000- amount in GON
+ status: `submitted` - three categories of status; submitted, confirmed, rejected

### Transaction
+ _links 
    + account
        + href: `/api/v1/accounts/GDIRF4UWPACXPPI4GW7CMTACTCNDIKJEHZK44RITZB4TD3YUM6CCVNGJ`
    + operations
        + href: `/api/v1/transactions/ghf6msRhE4jRf5DPib9UHD1msadvmZs9o53V9FQTb11/operations{?cursor,limit,order}`
        + templated: true
    + self
        + href: `/api/v1/transactions/`
+ block: 
+ created: `2018-09-12T09:08:35.157472400Z` - Created time of the transaction. It is set by wallet
+ fee: `10000` (string) - The fee paid by the source account
+ hash: `ghf6msRhE4jRf5DPib9UHD1msadvmZs9o53V9FQTb11` (string,required) - Hash of transaction. //TODO: link for the details
+ operation_count: 1 (number) - The number of operations in this transaction.
+ sequence_id: 0 (number) - the Sequence number of the source account.
+ source: `GDIRF4UWPACXPPI4GW7CMTACTCNDIKJEHZK44RITZB4TD3YUM6CCVNGJ` (string) -
        
### Operation
+ _embedded
    + records:(array)
        + (object):
            + _links:
                + self
                    + href: `/api/v1/operations/F6SEv2QhgwZwxUARbRacxyZaufzcTxdYDXJBpvf7pNAj-7nLuyg8radTExzBM2WhG37AwohBwEySBw4vj2xdtdjAs`
                + transaction
                    + href: `/api/v1/transactions/7nLuyg8radTExzBM2WhG37AwohBwEySBw4vj2xdtdjAs` 
        + block_height: 241,
        + body
            + target: GDEPYGGALPJ5HENXCNOQJPPDOQMA2YAXPERZ4XEAKVFFJJEVP4ZBK6QI - The funded account's public key
            + amount: `1000000000000` - amount in GON
        
        + confirmed: 
        + hash: F6SEv2QhgwZwxUARbRacxyZaufzcTxdYDXJBpvf7pNAj-7nLuyg8radTExzBM2WhG37AwohBwEySBw4vj2xdtdjAs - Hash of operation
        + proposed_time:
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

### Blocks
+ _embedded
    + records (array)
        + (object):
            + _links
                + self
                    + href: `/api/v1/blocks/AcFpZMr6EhxBuCw3xADUzepa395wmh3c5fo2cyxYCi1q`
        + confirmed: 2018-11-18T18:44:47.900933000+09:00
        + hash: `AcFpZMr6EhxBuCw3xADUzepa395wmh3c5fo2cyxYCi1q`
        + height: 1
        + prev_block_hash: `J8TQCCtsiLcRZpYtVN3ozCFByd24fjXe2BgodLkeXN7S`,
        + proposed_time: `2018-04-17T5:07:31.000000000Z`
        + proposer: `GDIRF4UWPACXPPI4GW7CMTACTCNDIKJEHZK44RITZB4TD3YUM6CCVNGJ`
        + proposer_transaction: `EQNSFnhzzz3bDpaZQekWPPNtr3kmRs5fUafBYAkHGXRP`
        + round: 0,
        + transactions (array)
            + `BivUS2tYjm1ZYXZNvKqRDa1eyBRTcE3DeuEDJVtuwNcm`
        + transactions_root: `BR2gsNw5WGjZ6HFPNr8fFAQPu42dqk1P7VVV7p5Efnru`
        + version: 0
+ _links
    + next
        + href: `/api/v1/blocks?cursor=1&limit=100&reverse=false`
    + prev
        + href: `/api/v1/blocks?cursor=1&limit=100&reverse=true`
    + self
        + href: `/api/v1/blocks`

### Block
+ _links
    + self
        + href: `/api/v1/blocks/AcFpZMr6EhxBuCw3xADUzepa395wmh3c5fo2cyxYCi1q`
+ confirmed: 2018-11-18T18:44:47.900933000+09:00
+ hash: `AcFpZMr6EhxBuCw3xADUzepa395wmh3c5fo2cyxYCi1q`
+ height: 3 
+ prev_block_hash: `J8TQCCtsiLcRZpYtVN3ozCFByd24fjXe2BgodLkeXN7S`,
+ proposed_time: `2018-04-17T5:07:31.000000000Z`
+ proposer: `GDIRF4UWPACXPPI4GW7CMTACTCNDIKJEHZK44RITZB4TD3YUM6CCVNGJ`
+ proposer_transaction: `EQNSFnhzzz3bDpaZQekWPPNtr3kmRs5fUafBYAkHGXRP`
+ round: 0
+ transactions (array)
    + `BivUS2tYjm1ZYXZNvKqRDa1eyBRTcE3DeuEDJVtuwNcm`
+ transactions_root: `BR2gsNw5WGjZ6HFPNr8fFAQPu42dqk1P7VVV7p5Efnru`
+ version: 0

### Problem
+ status:  500 (number)
+ title: `problem error message`
+ type: `https://boscoin.io/sebak/error/{error_code}`


### Problem NotFound
+ status: 400 (number)
+ title: `does not exists` 
+ type: `https://boscoin.io/sebak/error/128`
