# Group Trasactions
Transactions API


## Transactions [/v1/transactions]


### Payment transaction  [POST]
//TODO: How to make a transaction and sign

+ Request (application/json)
    
    + Attributes (Transaction Payment)

+ Response 200 (application/json; charset=utf-8)
    
    + Attributes (Transaction Payment)

+ Response 500 (application/problem+json; charset=utf-8)

    + Attributes (Problem)

## Transactions [/v1/transactions?limit={limit}&reverse={reverse}&cursor={cursor}]
+ Parameters
    
    + limit: `100` (integer, optional)
    
    + reverse: `false` (string, optional)
    
    + cursor: `` (string, optional)

### Retrieve transactions [GET]
<p> Retrieve all valid transactions </p>

<p> Streaming mode supported with header "Accept": "text/event-stream" </p>

+ Response 200 (application/hal+json; charset=utf-8)

    + Attributes (Transactions)

+ Response 500 (application/problem+json; charset=utf-8)

    + Attributes (Problem)

## Transaction [/v1/transactions/{hash}?limit={limit}&reverse={reverse}&cursor={cursor}]

+ Parameters
    
    + hash: `ghf6msRhE4jRf5DPib9UHD1msadvmZs9o53V9FQTb11` (string,required) - tx's hash
    
    + limit: `100` (integer, optional)
    
    + reverse: `false` (string, optional)
    
    + cursor: `` (string, optional)
    
### Get Transaction [GET]

+ Response 200 (application/hal+json; charset=utf-8)

    + Attributes (Transaction)

+ Response 500 (application/problem+json; charset=utf-8)

    + Attributes (Problem)

## Operations for Trasaction [/v1/transactions/{hash}/operations?limit={limit}&reverse={reverse}&cursor={cursor}]

+ Parameters
    
    + hash: `ghf6msRhE4jRf5DPib9UHD1msadvmZs9o53V9FQTb11` (string,required) - Transaction hash
    
    + limit: `100` (integer, optional)
        
    + reverse: `false` (string, optional)
        
    + cursor: `` (string, optional)

#### Get operations of transaction [GET]

+ Response 200 (application/hal+json; charset=utf-8)
    
    + Attributes (Operations)

+ Response 500 (application/problem+json; charset=utf-8)

    + Attributes (Problem)

