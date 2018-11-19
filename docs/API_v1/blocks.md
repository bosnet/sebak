# Group Blocks
Blocks API

## Blocks [/api/v1/blocks?cursor={cursor}&limit={limit}&reverse={reverse}] 

+ Parameters
    + cursor: `1207` (string, optional)  - a block height as cursor
    + reverse: `false` (string, optional)
    + limit: `100` (integer, optional)

### Retrieve blocks [GET]

<p>Retrieve all valid blocks </p>

<p> Streaming mode supported with header "Accept": "text/event-stream" </p>

+ Response 200 (application/hal+json; charset=utf-8)
    + Attributes (Blocks)

+ Response 500 (application/problem+json; charset=utf-8)
    + Attributes (Problem)


## Block Details [/api/v1/blocks/{hashOrHeight}]

+ Parameters
    + hashOrHeight: `CLNes5kkg7ozgnHBhpBXHMHFtPKo7z4RF8NZpNGRUB4i` `1207` (string,required) - a block hash or height

### Retrieve a block [GET]

<p> Retrieve a block by the hash or height <p>

+ Response 200 (application/hal+json; charset=utf-8)
    + Attributes (Block)
+ Response 404 (application/problem+json; charset=utf-8)
    + Attributes (Problem NotFound)
+ Response 500 (application/problem+json; charset=utf-8)
    + Attributes (Problem)
 
