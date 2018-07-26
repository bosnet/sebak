package node

import (
	"boscoin.io/sebak/pkg/support/logger"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

type Node struct {
	logger *logger.Logger
	config *Config
	server *server
}

func NewNode(config *Config) *Node {
	if config.ChainDbDir == "" {
		config.ChainDbDir = fmt.Sprintf("%s/chain", config.DataDir)
	}
	server := newServer(config)

	return &Node{
		logger: logger.NewLogger("node"),
		config: config,
		server: server,
	}
}

func (o *Node) RunForever() {
	o.start()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		for sig := range c {
			fmt.Printf("captured %v, exiting...\n", sig)
			os.Exit(1)
		}
	}()

	select {}
}

func (o *Node) start() {
	go o.server.Start()
}

func (o *Node) stop() {
}
