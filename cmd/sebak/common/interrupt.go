package cmdcommon

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func Interrupt(cancel <-chan struct{}) error {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	select {
	case sig := <-c:
		fmt.Println("Received signal ", sig, ", shutting down...")
		return nil
	case <-cancel:
		return errors.New("canceled")
	}
}
