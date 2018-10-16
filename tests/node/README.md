# SEBAK Integration tests

## Dependencies

This testsuite depends on `bash`, `curl`, `jq` and `docker`.
You should have the ability to build and image to run it.

## Layout

Integration tests reside in this directory, while unittests reside in the source folder as is standard Golang practice.
Each subdirectory contains an integration test.

## Requests

The base unit in a test is a request, which are represented by `json` files.
A name should follow the following naming convention:
- The name should be`request_$ID_[$NODE[_$DESCRIPTION]].json`
- $ID is an integer, allowing to order requests
- If two requests with the same `$ID` exists, they will be send at the same time
- $NODE is the port of the node to send this request to
- If $NODE is not provided, the request will be sent to all nodes
- $DESCRIPTION is just an optional human readable description

## Configuration

In the future, it would be sensible to provide a way to configure the network.
At the moment, the network is hard-coded to be the 3 nodes provided in this repository.
