## Data Structures

### Account
+ account_id: GDMZMF2EAK4E6NSZNSCJQQHQGMAOZ6UI3XQVVLMEJRFDPYHLY7PPHKLP (string, required)
+ balance: 10000000000000000000 (string,required)
+ sequence_id: 0 (number,required)
+ _links: 
    + operations:
        + href: `/accounts/GDMZMF2EAK4E6NSZNSCJQQHQGMAOZ6UI3XQVVLMEJRFDPYHLY7PPHKLP/operations{?cursor,limit,order}`
        + templated: true (boolean)
    + self:
        + href: `/accounts/GDMZMF2EAK4E6NSZNSCJQQHQGMAOZ6UI3XQVVLMEJRFDPYHLY7PPHKLP` (string)
    + transactions:
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
+ _links: 
    + accounts:
        + href: `/accounts/GDIRF4UWPACXPPI4GW7CMTACTCNDIKJEHZK44RITZB4TD3YUM6CCVNGJ`
    + operations:
        + href: `/transactions/ghf6msRhE4jRf5DPib9UHD1msadvmZs9o53V9FQTb11/operations{?cursor,limit,order}`
    + self:
        + href: `/transactions/ghf6msRhE4jRf5DPib9UHD1msadvmZs9o53V9FQTb11`


### Transactions
+ _embedded:
    + records:
        + _links:
            + accounts:
                + href: `/accounts/GDIRF4UWPACXPPI4GW7CMTACTCNDIKJEHZK44RITZB4TD3YUM6CCVNGJ`
            + operations:
                + href: `/transactions/ghf6msRhE4jRf5DPib9UHD1msadvmZs9o53V9FQTb11/operations{?cursor,limit,order}`
                + templated: true
            + self:
                + href: `/transactions/ghf6msRhE4jRf5DPib9UHD1msadvmZs9o53V9FQTb11`
        + id: ghf6msRhE4jRf5DPib9UHD1msadvmZs9o53V9FQTb11
        + hash:  ghf6msRhE4jRf5DPib9UHD1msadvmZs9o53V9FQTb11
        + account: GDVSXU343JMRBXGW3F5WLRMH6L6HFZ6IYMVMFSDUDJPNTXUGNOXC2R5Y
        + fee_paid: `10000`
        + operation_count: 1
        + sequence_id: 0
+ _links:
    + next:
        + href: /account/{address}/transactions
    + prev:
        + href: /account/{address}/transactions
    + self:
        + href: /account/{address}/transactions

### Operations
+ _embedded:
    + records:
        + _links:
            + self:
                + href: /operations/E4qTH5UmzHy2Psdxh8RaQomqJb1gcUZFVENimzV9YB8D-ghf6msRhE4jRf5DPib9UHD1msadvmZs9o53V9FQTb11
            + transactions:
                + href: /transactions/ghf6msRhE4jRf5DPib9UHD1msadvmZs9o53V9FQTb11
        + account: GDMZMF2EAK4E6NSZNSCJQQHQGMAOZ6UI3XQVVLMEJRFDPYHLY7PPHKLP
        + amount: 1000000
        + funder: GDPWP7BOOMEKK6DUQELQD7H5NEENPLDTQQWYOIBSFS65WH7DNG7UWVKB
        + hash: E4qTH5UmzHy2Psdxh8RaQomqJb1gcUZFVENimzV9YB8D
        + id: E4qTH5UmzHy2Psdxh8RaQomqJb1gcUZFVENimzV9YB8D
        + type: create-account
+ _links:
    + next:
        + href: /account/{address}/operations
    + prev:
        + href: /account/{address}/operations
    + self:
        + href: /account/{address}/operations


### Problem
+ type: `type`
+ title: `title`
+ detail: `detail`
+ instance: `instance`

