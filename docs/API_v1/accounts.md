# Group Accounts
Account API

## Account Details [/api/v1/accounts/{address}]
<p> In the BOScoin network, users interact by using accounts </p>

+ Parameters

    + address: `GDEPYGGALPJ5HENXCNOQJPPDOQMA2YAXPERZ4XEAKVFFJJEVP4ZBK6QI` (string, required) - a public address

### Retrieve an account [GET]
<p> Retrieve an account by the address </p>

+ Response 200 (application/hal+json; charset=utf-8)

    + Attributes (Account)

+ Response 404 (application/problem+json; charset=utf-8)

    + Attributes (Problem NotFound)

+ Response 500 (application/problem+json; charset=utf-8)
    
    + Attributes (Problem)
    

## Transactions for Account [/api/v1/accounts/{address}/transactions?limit={limit}&reverse={reverse}&cursor={cursor}]

+ Parameters

    + address: `GDIRF4UWPACXPPI4GW7CMTACTCNDIKJEHZK44RITZB4TD3YUM6CCVNGJ` (string, required) - a public address
    
    + limit: `100` (integer, optional)
            
    + reverse: `false` (string, optional)
            
    + cursor: `` (string, optional)


### List All Transactions for Account [GET]
<p> Retrieve all valid transactions that affected by the account </p>

<p> Streaming mode supported with header "Accept": "text/event-stream" </p>

+ Response 200 (application/hal+json; charset=utf-8)

    + Attributes (Transactions)

+ Response 500 (application/problem+json; charset=utf-8)
    
    + Attributes (Problem)


## Operations for Account [/api/v1/accounts/{address}/operations?limit={limit}&reverse={reverse}&cursor={cursor}]
<p> Retrieve all operations that were included in valid transactions that affected by the account </p>

<p> Streaming mode supported with header "Accept": "text/event-stream" </p>

+ Parameters

    + address: `GDVSXU343JMRBXGW3F5WLRMH6L6HFZ6IYMVMFSDUDJPNTXUGNOXC2R5Y` (string, required) - a public address

    + limit: `100` (integer, optional)
        
    + reverse: `false` (string, optional)
        
    + cursor: `` (string, optional)

### List All Operations for Account [GET]

+ Response 200 (application/hal+json; charset=utf-8)

    + Attributes (Operation)

+ Response 500 (application/problem+json; charset=utf-8)

    + Attributes (Problem)