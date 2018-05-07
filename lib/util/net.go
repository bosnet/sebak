package util

import (
	"net"
	"strconv"
)

func CheckPortInUse(port int) error {
	_, err := net.DialTimeout("tcp", net.JoinHostPort("", strconv.FormatInt(int64(port), 10)), 10)
	return err
}
