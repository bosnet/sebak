# Group Accounts
Account API

## Account Details [/v1/account/{address}]

+ Parameters

    + address: `GDVSXU343JMRBXGW3F5WLRMH6L6HFZ6IYMVMFSDUDJPNTXUGNOXC2R5Y` (string, required) - a public address

### Retrieve an account [GET]
Retrieve an account by the address

+ Response 200 (application/hal+json; charset=utf-8)

    + Attributes (Account)

+ Response 404 (application/problem+json; charset=utf-8)

    + Attributes (Problem NotFound)

+ Response 500 (application/problem+json; charset=utf-8)
    
    + Attributes (Problem)
    

## Transactions for Account [/v1/account/{address}/transactions]

+ Parameters

    + address: `GDVSXU343JMRBXGW3F5WLRMH6L6HFZ6IYMVMFSDUDJPNTXUGNOXC2R5Y` (string, required) - a public address


### List All Transactions for Account [GET]

+ Response 200 (application/hal+json; charset=utf-8)

    + Attributes (Transactions)

+ Response 500 (application/problem+json; charset=utf-8)
    
    + Attributes (Problem)


## Operations for Account [/v1/account/{address}/operations]

+ Parameters

    + address: `GDVSXU343JMRBXGW3F5WLRMH6L6HFZ6IYMVMFSDUDJPNTXUGNOXC2R5Y` (string, required) - a public address

### List All Operations for Account [GET]

+ Response 200 (application/hal+json; charset=utf-8)

    + Attributes (Operations)

+ Response 500 (application/problem+json; charset=utf-8)

    + Attributes (Problem)

