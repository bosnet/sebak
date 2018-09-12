# Group Trasactions
Transactions API


## Transactions [/v1/transactions]

### Post transaction [POST]

+ Request (application/json)
    
    + Attributes (Transaction)

+ Response 200 (application/hal+json; charset=utf-8)

+ Response 500 (application/problem+json; charset=utf-8)

    + Attributes (Problem)

### Retrieve transactions [GET]

+ Response 200 (application/hal+json; charset=utf-8)

+ Response 500 (application/problem+json; charset=utf-8)

    + Attributes (Problem)

## Transaction [/v1/transactions/{hash}]

+ Parameters
    
    + hash: `ghf6msRhE4jRf5DPib9UHD1msadvmZs9o53V9FQTb11` (string,required) - tx's hash

### Get Transaction [GET]

+ Response 200 (application/hal+json; charset=utf-8)

    + Attributes (Transaction)

+ Response 500 (application/problem+json; charset=utf-8)

    + Attributes (Problem)

