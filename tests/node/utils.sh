# Source me

function die () # string message
{
    MESSAGE=${1:-"An error happened, but no message was provided"}
    echo 1>&2 "[${TEST_NAME:-'Unknown test'}] Error: ${MESSAGE}"
    exit 1
}
