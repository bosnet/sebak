# SEBAK Integration tests

## Dependencies

This testsuite depends on `bash`, `curl`, `jq` and `docker`.
You should have the ability to build and image to run it.

## Layout

Integration tests that require a long time(more than 2 minutes) reside in this directory.
Therefore, these should be seperated with the other tests and done with parallel.

## Configuration

At the moment, the network is hard-coded to be the 4 nodes provided in this repository.
