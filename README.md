# SEBAK

[![GoDoc](https://godoc.org/github.com/golang/gddo?status.svg)](http://godoc.org/github.com/spikeekips/sebak/lib) [![Build Status](https://travis-ci.org/spikeekips/sebak.svg?branch=master)](https://travis-ci.org/spikeekips/sebak)

Sebak is the core node for crypto-currency blockchain.

## Installation

To start sebak, install Go 1.8 or above and run `go get`:

```
$ go get github.com/spikeekips/sebak/cmd/sebak
```

## Test

You can test sebak. Before testing, you must install 'dep'. You can check how to install 'dep' in [dep installation](https://github.com/golang/dep#installation).

```
$ mkdir sebak
$ cd sebak
$ export GOPATH=$(pwd)
$ go get github.com/spikeekips/sebak
$ cd src/github.com/spikeekips/sebak
$ dep ensure
```

It's ready to test.

```
$ go test ./...
```

You can see the detailed logs:
```
$ go test ./... -v
```
