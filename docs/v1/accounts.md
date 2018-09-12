# Group Accounts
Account API

## Account Details [/v1/account/{address}]

+ Parameters

    + address: `GDVSXU343JMRBXGW3F5WLRMH6L6HFZ6IYMVMFSDUDJPNTXUGNOXC2R5Y` (string, required) - a public address

### Retrieve an account [GET]
Retrieve an account by the address

+ Response 200 (application/json; charset=utf-8)

       + Attributes (Account)

+ Response 404 (text/plain; charset=utf-8)

+ Response 500 (application/json; charset=utf-8)


## Transactions for Account [/v1/account/{address}/transactions]

+ Parameters

    + address: `GDVSXU343JMRBXGW3F5WLRMH6L6HFZ6IYMVMFSDUDJPNTXUGNOXC2R5Y` (string, required) - a public address


### List All Trasactions for Account [GET]


+ Response 200 (application/json; charset=utf-8)

+ Response 500 (application/json; charset=utf-8)


## Operations for Account [/v1/account/{address}/operations]

+ Parameters

    + address: `GDVSXU343JMRBXGW3F5WLRMH6L6HFZ6IYMVMFSDUDJPNTXUGNOXC2R5Y` (string, required) - a public address

### List All Operations for Account [GET]

+ Response 200 (application/json; charset=utf-8)


