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
    
    + hash: `hash` (string,required) - tx's hash

### Get Transaction [GET]

+ Response 200 (application/hal+json; charset=utf-8)

+ Response 500 (application/problem+json; charset=utf-8)

    + Attributes (Problem)

