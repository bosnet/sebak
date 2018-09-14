## Data Structures

### Account
+ account_id: GDMZMF2EAK4E6NSZNSCJQQHQGMAOZ6UI3XQVVLMEJRFDPYHLY7PPHKLP (string, required)
+ balance: 10000000000000000000 (string,required)
+ sequence_id: 0 (number,required)
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
+ id: `ghf6msRhE4jRf5DPib9UHD1msadvmZs9o53V9FQTb11` (string,required)
+ hash: `ghf6msRhE4jRf5DPib9UHD1msadvmZs9o53V9FQTb11` (string,required)
+ account: `GDIRF4UWPACXPPI4GW7CMTACTCNDIKJEHZK44RITZB4TD3YUM6CCVNGJ` (string,required) 
+ fee_paid: `10000` (string,required)
+ sequence_id: 0 (number)
+ created_at: `2018-09-12T09:08:35.157472400Z` 
+ operation_count: 1 (number)
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
    + created: `2018-01-01T00:00:00.000000000Z`
    + hash: `2g3ZSrEnsUWeX5Mxz5uTh2b4KVpVQS7Ek2HzZd759FHn`
    + signature: `3oWmCMNHExRQnZVEBSH16ZBgLE6ayz7t1fsjzTjAB6WpXMpkDJbhcL8KudqFFG21XmfSXnJH1BLhnBUh4p68yFeR`
+ B
    + source: GDIRF4UWPACXPPI4GW7CMTACTCNDIKJEHZK44RITZB4TD3YUM6CCVNGJ
    + fee: 10000
    + sequenceID: 1
    + operations (array):
        + (object):
            + H 
                + type: "payment"
            + B
                + target: GDTEPFWEITKFHSUO44NQABY2XHRBBH2UBVGJ2ZJPDREIOL2F6RAEBJE4
                + amount: 100000000

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
        + id: ghf6msRhE4jRf5DPib9UHD1msadvmZs9o53V9FQTb11
        + hash:  ghf6msRhE4jRf5DPib9UHD1msadvmZs9o53V9FQTb11
        + account: GDIRF4UWPACXPPI4GW7CMTACTCNDIKJEHZK44RITZB4TD3YUM6CCVNGJ
        + fee_paid: `10000`
        + operation_count: 1
        + sequence_id: 0
+ _links
    + next
        + href: /account/GDIRF4UWPACXPPI4GW7CMTACTCNDIKJEHZK44RITZB4TD3YUM6CCVNGJ/transactions
    + prev
        + href: /account/GDIRF4UWPACXPPI4GW7CMTACTCNDIKJEHZK44RITZB4TD3YUM6CCVNGJ/transactions
    + self
        + href: /account/GDIRF4UWPACXPPI4GW7CMTACTCNDIKJEHZK44RITZB4TD3YUM6CCVNGJ/transactions

### Operations
+ _embedded
    + records
        + _links
            + self
                + href: /operations/E4qTH5UmzHy2Psdxh8RaQomqJb1gcUZFVENimzV9YB8D-ghf6msRhE4jRf5DPib9UHD1msadvmZs9o53V9FQTb11
            + transactions
                + href: /transactions/ghf6msRhE4jRf5DPib9UHD1msadvmZs9o53V9FQTb11
        + account: GDIRF4UWPACXPPI4GW7CMTACTCNDIKJEHZK44RITZB4TD3YUM6CCVNGJ
        + amount: 1000000
        + funder: GDPWP7BOOMEKK6DUQELQD7H5NEENPLDTQQWYOIBSFS65WH7DNG7UWVKB
        + hash: E4qTH5UmzHy2Psdxh8RaQomqJb1gcUZFVENimzV9YB8D
        + id: E4qTH5UmzHy2Psdxh8RaQomqJb1gcUZFVENimzV9YB8D
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
+ title: `problem error message `
+ type: `https://boscoin.io/sebak/error/{error_code}`


### Problem NotFound
+ status: 400 (number)
+ title: `does not exists` 
+ type: `https://boscoin.io/sebak/error/128`
