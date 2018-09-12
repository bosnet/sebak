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
+ Hash: hash (string,required)
