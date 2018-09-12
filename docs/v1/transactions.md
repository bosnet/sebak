# Group Trasactions
Transactions API


## Transactions [/v1/transactions]

### Post transaction [POST]

+ Request (application/json)
    
    + Attributes (Transaction)

+ Response 200 (application/json; charset=utf-8)

### Retrieve transactions [GET]

+ Response 200 (application/json; charset=utf-8)


## Transaction [/v1/transactions/{hash}]

+ Parameters
    
    + hash: `hash` (string,required) - tx's hash

### Get Transaction [GET]

+ Response 200 (application/json; charset=utf-8)
