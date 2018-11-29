## Paging

Pages represent a subset of a larger collection of objects. The SEBAK HTTP API utilizes `cursoring` to paginate large result sets. Cursoring separates results into pages 

<h3> Cursor </h3>

A `cursor` is a point to a specific location in resources. 

<h3> Embedded Resources </h3>

A page containts an embedded set of `records`, regardless of the contained resource.

<h4> Links </h4> 

|        | Example                                                |  Relation                          | 
|--------|--------------------------------------------------------|------------------------------------|
| Self   | `/transactions`                                        |                                    | 
| Prev   | `/transactions?cursor={cursor}&reverse=true&limit=10`  | The previous page of results       |
| Next   | `/transactions?cursor={cursor}&reverse=false&limit=10` | The next page of results           |


