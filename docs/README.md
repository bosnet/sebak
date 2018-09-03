# API documents with [API blueprint](https://apiblueprint.org/)

Install [snowboard](https://github.com/bukalapak/snowboard).

Generate HTML document:

```
$ snowboard html -o output API.md
```

Serve HTML document:

```
$ snowboard html -o output -s API.md
```

Open <http://localhost:8088>.

Validate API document:

```
$ snowboard lint API.md
```
