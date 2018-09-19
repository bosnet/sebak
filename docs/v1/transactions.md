# Group Trasactions
Transactions API


## Transactions [/v1/transactions]

### Payment transaction  [POST]

+ Request (application/json)
    
    + Attributes (Transaction Payment)

+ Response 200 (application/json; charset=utf-8)
    
    + Attributes (Transaction Payment)

+ Response 500 (application/problem+json; charset=utf-8)

    + Attributes (Problem)

### Retrieve transactions [GET]

+ Response 200 (application/hal+json; charset=utf-8)

    + Attributes (Transactions)

+ Response 500 (application/problem+json; charset=utf-8)

    + Attributes (Problem)

## Transaction [/v1/transactions/{hash}]

+ Parameters
    
    + hash: `ghf6msRhE4jRf5DPib9UHD1msadvmZs9o53V9FQTb11` (string,required) - Transaction's hash

### Get Transaction [GET]

+ Response 200 (application/hal+json; charset=utf-8)

    + Attributes (Transaction)

+ Response 500 (application/problem+json; charset=utf-8)

    + Attributes (Problem)

## Operations for Trasaction [/v1/transactions/{hash}/operations]

+ Parameters
    
    + hash: `ghf6msRhE4jRf5DPib9UHD1msadvmZs9o53V9FQTb11` (string,required) - Transaction's hash

#### Get operations of transaction [GET]

+ Response 200 (application/hal+json; charset=utf-8)
    
    + Attributes (Operations)

+ Response 500 (application/problem+json; charset=utf-8)

    + Attributes (Problem)

