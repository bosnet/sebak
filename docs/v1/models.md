## Data Structures

### Account
+ address: GDMZMF2EAK4E6NSZNSCJQQHQGMAOZ6UI3XQVVLMEJRFDPYHLY7PPHKLP (string, required) - The account’s public key encoded into a base32 string representation.
+ balance: 10000000000000000000 (string,required) - GON. 1 BOS = 10,000,000 GON
+ sequence_id: 0 (number,required) - The Current sequence number. It needed to submitting a transaction from this account
+ _links 
    + operations
        + href: `/accounts/GDMZMF2EAK4E6NSZNSCJQQHQGMAOZ6UI3XQVVLMEJRFDPYHLY7PPHKLP/operations{?cursor,limit,order}`
        + templated: true (boolean)
    + self
        + href: `/accounts/GDMZMF2EAK4E6NSZNSCJQQHQGMAOZ6UI3XQVVLMEJRFDPYHLY7PPHKLP` (string)
    + transactions
        + href: `/accounts/GDMZMF2EAK4E6NSZNSCJQQHQGMAOZ6UI3XQVVLMEJRFDPYHLY7PPHKLP/transactions{?cursor,limit,order}` 
        + templated: true (boolean)

### Transaction
+ hash: `ghf6msRhE4jRf5DPib9UHD1msadvmZs9o53V9FQTb11` (string,required) - Hash of transaction. //TODO: link for the details
+ source: `GDIRF4UWPACXPPI4GW7CMTACTCNDIKJEHZK44RITZB4TD3YUM6CCVNGJ` (string,required) - 
+ fee: `10000` (string,required) - The fee paid by the source account
+ sequence_id: 0 (number) - the Sequence number of the source account 
+ created: `2018-09-12T09:08:35.157472400Z` - Created time of the transaction. It is set by wallet
+ operation_count: 1 (number) - The number of operations in this transaction.
+ _links 
    + accounts
        + href: `/accounts/GDIRF4UWPACXPPI4GW7CMTACTCNDIKJEHZK44RITZB4TD3YUM6CCVNGJ`
    + operations
        + href: `/transactions/ghf6msRhE4jRf5DPib9UHD1msadvmZs9o53V9FQTb11/operations{?cursor,limit,order}`
    + self
        + href: `/transactions/ghf6msRhE4jRf5DPib9UHD1msadvmZs9o53V9FQTb11`


### Transaction Payment
+ T: transaction
+ H 
    + version: `` - Transaction version
    + created: `2018-01-01T00:00:00.000000000Z` - Created time of the transaction.
    + hash: `2g3ZSrEnsUWeX5Mxz5uTh2b4KVpVQS7Ek2HzZd759FHn` - Hash of the transaction body.
    + signature: `3oWmCMNHExRQnZVEBSH16ZBgLE6ayz7t1fsjzTjAB6WpXMpkDJbhcL8KudqFFG21XmfSXnJH1BLhnBUh4p68yFeR` - Signature signed by source account
+ B
    + source: GDIRF4UWPACXPPI4GW7CMTACTCNDIKJEHZK44RITZB4TD3YUM6CCVNGJ - Source account
    + fee: 10000 - The fee paid by the source account for this transaction. Minimum is 10000 GON
    + sequenceID: 1 - The last sequence number of the source account
    + operations (array):
        + (object):
            + H 
                + type: "payment" - operation type. ex. payment, create-account
            + B
                + target: GDTEPFWEITKFHSUO44NQABY2XHRBBH2UBVGJ2ZJPDREIOL2F6RAEBJE4 - The funded account's public key
                + amount: 100000000 - amount in GON

### Transactions
+ _embedded
    + records
        + _links
            + accounts
                + href: `/accounts/GDIRF4UWPACXPPI4GW7CMTACTCNDIKJEHZK44RITZB4TD3YUM6CCVNGJ`
            + operations
                + href: `/transactions/ghf6msRhE4jRf5DPib9UHD1msadvmZs9o53V9FQTb11/operations{?cursor,limit,order}`
                + templated: true
            + self
                + href: `/transactions/ghf6msRhE4jRf5DPib9UHD1msadvmZs9o53V9FQTb11`
        + hash:  ghf6msRhE4jRf5DPib9UHD1msadvmZs9o53V9FQTb11
        + account: GDIRF4UWPACXPPI4GW7CMTACTCNDIKJEHZK44RITZB4TD3YUM6CCVNGJ
        + fee: `10000`
        + operation_count: 1
        + sequence_id: 0
+ _links
    + next
        + href: /account/GDIRF4UWPACXPPI4GW7CMTACTCNDIKJEHZK44RITZB4TD3YUM6CCVNGJ/transactions
    + prev
        + href: /account/GDIRF4UWPACXPPI4GW7CMTACTCNDIKJEHZK44RITZB4TD3YUM6CCVNGJ/transactions
    + self
        + href: /account/GDIRF4UWPACXPPI4GW7CMTACTCNDIKJEHZK44RITZB4TD3YUM6CCVNGJ/transactions
        
### Operation
+ source: GDIRF4UWPACXPPI4GW7CMTACTCNDIKJEHZK44RITZB4TD3YUM6CCVNGJ - Source account
+ amount: 1000000 - amount in GON
+ target: GDPWP7BOOMEKK6DUQELQD7H5NEENPLDTQQWYOIBSFS65WH7DNG7UWVKB - The funded account's public key
+ hash: E4qTH5UmzHy2Psdxh8RaQomqJb1gcUZFVENimzV9YB8D - Hash of operation
+ type: create-account - operation type. ex. payment, create-account
+ _links 
    + accounts 
        + href: `/accounts/GDIRF4UWPACXPPI4GW7CMTACTCNDIKJEHZK44RITZB4TD3YUM6CCVNGJ`
    + transaction 
        + href: `/operations/E4qTH5UmzHy2Psdxh8RaQomqJb1gcUZFVENimzV9YB8D/transactions`
    + self
        + href: `/operations/E4qTH5UmzHy2Psdxh8RaQomqJb1gcUZFVENimzV9YB8D`

### Operations
+ _embedded
    + records
        + _links
            + self
                + href: /operations/E4qTH5UmzHy2Psdxh8RaQomqJb1gcUZFVENimzV9YB8D-ghf6msRhE4jRf5DPib9UHD1msadvmZs9o53V9FQTb11
            + transactions
                + href: /transactions/ghf6msRhE4jRf5DPib9UHD1msadvmZs9o53V9FQTb11
        + source: GDIRF4UWPACXPPI4GW7CMTACTCNDIKJEHZK44RITZB4TD3YUM6CCVNGJ
        + amount: 1000000
        + target: GDPWP7BOOMEKK6DUQELQD7H5NEENPLDTQQWYOIBSFS65WH7DNG7UWVKB
        + hash: E4qTH5UmzHy2Psdxh8RaQomqJb1gcUZFVENimzV9YB8D
        + type: create-account
+ _links
    + next
        + href: /account/GDIRF4UWPACXPPI4GW7CMTACTCNDIKJEHZK44RITZB4TD3YUM6CCVNGJ/operations
    + prev
        + href: /account/GDIRF4UWPACXPPI4GW7CMTACTCNDIKJEHZK44RITZB4TD3YUM6CCVNGJ/operations
    + self
        + href: /account/GDIRF4UWPACXPPI4GW7CMTACTCNDIKJEHZK44RITZB4TD3YUM6CCVNGJ/operations


### Problem
+ status:  500 (number)
+ title: `problem error message`
+ type: `https://boscoin.io/sebak/error/{error_code}`


### Problem NotFound
+ status: 400 (number)
+ title: `does not exists` 
+ type: `https://boscoin.io/sebak/error/128`
