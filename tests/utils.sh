# Source me

function die () # string message
{
    MESSAGE=${1:-"An error happened, but no message was provided"}
    echo 1>&2 "[${TEST_NAME:-'Unknown test'}] Error: ${MESSAGE}"
    exit 1
}

function getAccount () # u16 port, string addr
{
    if [ $# -ne 2 ]; then
        die "2 arguments expected for getAccount, not $#"
    fi

    echo $(curl --insecure \
                  --request GET \
                  --header "Accept: application/json" \
                  https://127.0.0.1:${1}/api/account/${2} \
                  2>/dev/null)
}
