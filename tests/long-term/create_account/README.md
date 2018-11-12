# Account creation tests

Create 10 accounts with 10,000 BOS

1. Run the following(see `default_runner.sh`):
```
    curl --insecure \
         --request POST \
         --header "Content-Type: application/json" \
         --data "request_1_2821_create_10.json" \
         https://127.0.0.1:${PORT}/api/v1/transactions \
         >/dev/null 2>&1
```

1. Run `create_account.check`
