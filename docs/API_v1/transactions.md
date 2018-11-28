# Group Trasactions
Transactions API


## Transactions [/api/v1/transactions]


### Payment transaction  [POST] 

+ Data Body consist of 3 parts, ; T, H, B
    + T : ‘transaction’

    + H : H means Header. it consists of version, hash, signature & created.

        + Version means to transaction version. At the moment 1.
        + Hash means transaction hash.
        + signature is signed data from client.
            + How can you make signature? 
            
            Please check this link first. 
            
            You need 3 variables to make signature; RLPdata which is hashing, network id and source’s secret seed. 
            
            You can see that Which kinds of variables necessary. 
            
            You can use [JavaScript SDK] to make signature or [Python SDK]. Please check above SDKs.

        + created means to transcation created time.
    + B : B means Body.  It is RLP data.  so you have to encode B data to RLP format.   It contains; source , fee, sequence id, and operations.

        + source; means that public address which will BOScoin withdraw .
        + fee : data type is String.
        + sequence id
            + How can you get sequence id? 
            
            When you finished account creation, you can access http(or https)://{IP that you set up sebak node}/api/v1/accounts/{Public address that account you created}. 
            
            Then you can see sequence_id in response.
        + operations: It is json array consist of H & B. H include type, which means operation type. B include target & amount.
        + H : type ( type should be set ‘payment’ )
        + B : target ( Public address you want to send.) , amount ( amount data type is String .)
 
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
