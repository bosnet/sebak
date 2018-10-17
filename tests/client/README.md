# SEBAK Integration tests written in GoLang

## Dependencies

This testsuite requires `lib/client` module in the SEBAK repository.
Hence, the entire project should be built with this testsuite.
Please see the `Dockerfile`

## Layout

Files for tests in this directory should have postfix `_test.go`.
No submodule(sub directory) is permitted 

## Usage

If you want to add a test case then, add a file named `xxx_test.go`.
Write test code by using the client module.
Everything else is like GoLang `testing`

## Execute Specific Test Only
Execute `./run.sh [test_name]` where `test_name` is what you want to test.
