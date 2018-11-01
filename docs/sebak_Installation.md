## Prerequisite
Before installing, you must install Go 1.11 or above.

To start sebak:

```
$ # cd to any folder you see fit, and you might want to set GOPATH
$ git clone https://github.com/bosnet/sebak.git
$ cd sebak
$ go install ./...
$ sebak <command>
```

## Test

You can test sebak. see below.

```
$ go test ./...
```

You can see the detailed logs;
```
$ go test ./... -v
```
