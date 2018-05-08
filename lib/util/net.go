package util

import (
	"errors"
	"net"
	"strconv"
)

func CheckPortInUse(port int) error {
	_, err := net.DialTimeout("tcp", net.JoinHostPort("", strconv.FormatInt(int64(port), 10)), 10)
	return err
}

func CheckBindString(b string) error {
	_, port, err := net.SplitHostPort(b)
	if err != nil {
		return err
	}

	var portInt int64
	if portInt, err = strconv.ParseInt(port, 10, 64); err != nil {
		return err
	} else if portInt < 1 {
		return errors.New("invalid port")
	}

	return nil
}
