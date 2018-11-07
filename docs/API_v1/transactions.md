# Group Trasactions
Transactions API


## Transactions [/api/v1/transactions]


### Payment transaction  [POST] 
//TODO: How to make a transaction and sign 

+ Request (application/json)
    
    + Attributes (Transaction Payment)

+ Response 200 (application/json; charset=utf-8)
    
    + Attributes (Transaction Post)

+ Response 500 (application/problem+json; charset=utf-8)

    + Attributes (Problem)

## Transactions [/api/v1/transactions?limit={limit}&reverse={reverse}&cursor={cursor}]
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

## Transaction [/api/v1/transactions/{hash}]

+ Parameters
    
    + hash: `ghf6msRhE4jRf5DPib9UHD1msadvmZs9o53V9FQTb11` (string,required) - tx's hash
    
### Get Transaction [GET]
<p> Retrieve a transaction by transaction hash </p>

<p> Streaming mode supported with header "Accept": "text/event-stream" </p>

+ Response 200 (application/hal+json; charset=utf-8)

    + Attributes (Transaction)

+ Response 500 (application/problem+json; charset=utf-8)

    + Attributes (Problem)

## Operations for Trasaction [/api/v1/transactions/{hash}/operations?limit={limit}&reverse={reverse}&cursor={cursor}]

+ Parameters
    
    + hash: `ghf6msRhE4jRf5DPib9UHD1msadvmZs9o53V9FQTb11` (string,required) - Transaction hash
    
    + limit: `100` (integer, optional)
        
    + reverse: `false` (string, optional)
        
    + cursor: `` (string, optional)

#### Get operations of transaction [GET]

+ Response 200 (application/hal+json; charset=utf-8)
    
    + Attributes (Operation)

+ Response 500 (application/problem+json; charset=utf-8)

    + Attributes (Problem)

## History for Trasaction [/api/v1/transactions/{hash}/history?limit={limit}&reverse={reverse}&cursor={cursor}]

+ Parameters
    
    + hash: `7nLuyg8radTExzBM2WhG37AwohBwEySBw4vj2xdtdjAs` (string,required) - Transaction hash
    
    + limit: `100` (integer, optional)
        
    + reverse: `false` (string, optional)
        
    + cursor: `` (string, optional)

#### Get history of transaction [GET]

+ Response 200 (application/hal+json; charset=utf-8)
    
    + Attributes (TransactionHistory)

+ Response 500 (application/problem+json; charset=utf-8)

    + Attributes (Problem)