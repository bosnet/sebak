package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	logging "github.com/inconshreveable/log15"
	isatty "github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
	"github.com/stellar/go/keypair"

	"github.com/owlchain/sebak/cmd/sebak/common"
	"github.com/owlchain/sebak/lib/common"
)

const (
	basePort                  int    = 12345
	baseContainerPort         int    = 12000
	networkID                 string = "test sebak-network"
	dockerContainerNamePrefix string = "scn."
	nodeAliasFormat           string = "v%d"
)

const (
	defaultLogLevel        logging.Lvl = logging.LvlInfo
	defaultSebakLogLevel   logging.Lvl = logging.LvlDebug
	defaultDockerImageName string      = "boscoin/sebak/compose-network:latest"
)

var (
	cmd        *cobra.Command
	log        logging.Logger
	dockerHost string

	flagNumberOfNodes uint
	flagLogLevel      string = defaultLogLevel.String()
	flagSebakLogLevel string = defaultSebakLogLevel.String()
	flagImageName     string = defaultDockerImageName
	flagForceClean    bool   = false
)

func parseFlags() {
	{
		var err error
		var logLevel logging.Lvl
		if logLevel, err = logging.LvlFromString(flagLogLevel); err != nil {
			fmt.Printf("invalid `log-level`: %v\n", err)
			os.Exit(1)
		}

		var formatter logging.Format
		if isatty.IsTerminal(os.Stdout.Fd()) {
			formatter = logging.TerminalFormat()
		} else {
			formatter = logging.JsonFormatEx(false, true)
		}
		logHandler := logging.StreamHandler(os.Stdout, formatter)

		log = logging.New("module", "main")
		log.SetHandler(logging.LvlFilterHandler(logLevel, logHandler))
	}

	{
		var err error
		if _, err = logging.LvlFromString(flagSebakLogLevel); err != nil {
			fmt.Printf("invalid `sebak-log-level`: %v\n", err)
			os.Exit(1)
		}
	}

	log.Debug("Starting to compose sebak network")
	log.Debug(fmt.Sprintf(
		`
number of nodes: %d
      log level: %s
sebak log level: %s
		`,
		flagNumberOfNodes,
		flagLogLevel,
		flagSebakLogLevel,
	))
}

func cleanDocker(cli *client.Client) (err error) {
	ctx := context.Background()
	var cl []types.Container
	if cl, err = cli.ContainerList(ctx, types.ContainerListOptions{All: true}); err != nil {
		log.Error("failed to get container list", "error", err)
		return
	}

	for _, c := range cl {
		log.Debug("found container", "container", c)
		for _, name := range c.Names {
			if !strings.HasPrefix(name[1:], dockerContainerNamePrefix) {
				continue
			}

			// remove container :)
			if err = cli.ContainerRemove(ctx, c.ID, types.ContainerRemoveOptions{Force: true}); err != nil {
				log.Error("failed to remove container", "error", err, "container", c)
				return
			}
			log.Debug("container removed", "container", c)
		}
	}

	return
}

func runContainer(cli *client.Client, genesisKeypair *keypair.Full, node *sebakcommon.Validator) (err error) {
	ctx := context.Background()

	var images []types.ImageSummary
	if images, err = cli.ImageList(ctx, types.ImageListOptions{All: true}); err != nil {
		log.Error("failed to get container list", "error", err)
		return
	}
	if len(images) < 1 {
		err = errors.New("image not found")
		return
	}

	var imageID string
	for _, i := range images {
		if _, found := sebakcommon.InStringArray(i.RepoTags, flagImageName); !found {
			continue
		}
		imageID = i.ID
		break
	}

	if len(imageID) < 1 {
		err = fmt.Errorf("image not found")
		log.Error("failed to find the image for sebak compose network", "image", flagImageName, "error", err)
		return
	}

	var env_validators []string
	for _, v := range node.GetValidators() {
		s := fmt.Sprintf("%s,%s,%s", v.Address(), v.Endpoint(), v.Alias())
		env_validators = append(env_validators, s)
	}

	envs := []string{
		"SEBAK_TLS_CERT=/sebak.crt",
		"SEBAK_TLS_KEY=/sebak.key",
		fmt.Sprintf("SEBAK_LOG_LEVEL=%s", flagSebakLogLevel),
		fmt.Sprintf("SEBAK_SECRET_SEED=%s", node.Keypair().Seed()),
		fmt.Sprintf("SEBAK_NETWORK_ID=%s", networkID),
		fmt.Sprintf("SEBAK_ENDPOINT=%s", node.Endpoint()),
		fmt.Sprintf("SEBAK_GENESIS_BLOCK=%s", genesisKeypair.Seed()),
		fmt.Sprintf("SEBAK_VALIDATORS=%s", strings.Join(env_validators, " ")),
	}

	volumes := map[string]string{
		"/home/ubuntu/a/entrypoint.sh": "/entrypoint.sh",
	}
	var mounts []mount.Mount
	for s, t := range volumes {
		m := mount.Mount{Type: mount.TypeBind, Source: s, Target: t}
		mounts = append(mounts, m)
	}

	_, port := node.Endpoint().HostAndPort()
	containerConfig := &container.Config{
		Image:        imageID,
		AttachStdin:  false,
		AttachStdout: false,
		ExposedPorts: nat.PortSet{nat.Port(port): {}},
		Tty:          false,
		OpenStdin:    false,
		Entrypoint:   []string{"/bin/bash", "/entrypoint.sh"},
		Env:          envs,
	}
	containerHostConfig := &container.HostConfig{
		Mounts: mounts,
		PortBindings: nat.PortMap{
			nat.Port(fmt.Sprintf("%d/tcp", basePort)): []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: port,
				},
			},
		},
		NetworkMode: "host",
	}

	var containerBody container.ContainerCreateCreatedBody
	containerBody, err = cli.ContainerCreate(
		ctx,
		containerConfig,
		containerHostConfig,
		&network.NetworkingConfig{},
		fmt.Sprintf("%s%s", dockerContainerNamePrefix, node.Alias()),
	)
	if err != nil {
		log.Error("failed to create container", "error", err)
		return
	}

	if err = cli.ContainerStart(ctx, containerBody.ID, types.ContainerStartOptions{}); err != nil {
		log.Error("failed to start container", "error", err)
		return
	}

	return
}

func getDockerHost() (host string, err error) {
	dhost := os.Getenv("DOCKER_HOST")
	if dhost == "" {
		dhost = client.DefaultDockerHost
	}
	var addr string
	if _, addr, _, err = client.ParseHost(dhost); err != nil {
		log.Error("failed to parse docker host", "error", err, "host", dhost)
		return
	}

	host, _, err = net.SplitHostPort(addr)

	return
}

func composeNetwork() map[string]*sebakcommon.Validator {
	log.Debug("trying to compose network", "number of nodes", flagNumberOfNodes)

	var port int = baseContainerPort
	nodes := map[string]*sebakcommon.Validator{}
	for i := 0; i < int(flagNumberOfNodes); i++ {
		kp, _ := keypair.Random()
		endpoint, _ := sebakcommon.NewEndpointFromString(fmt.Sprintf(
			"https://%s:%d",
			dockerHost,
			port,
		))
		alias := fmt.Sprintf(nodeAliasFormat, i)

		node, _ := sebakcommon.NewValidator(kp.Address(), endpoint, alias)
		node.SetAlias(alias)
		node.SetKeypair(kp)
		nodes[kp.Address()] = node

		log.Debug(
			"generate node",
			"address", kp.Address(),
			"secret seed", kp.Seed(),
			"endpoint", endpoint,
			"alias", alias,
		)

		port += 1
	}

	for a0, n0 := range nodes {
		for a1, n1 := range nodes {
			if a0 == a1 {
				continue
			}
			n0.AddValidators(n1)
		}
	}

	return nodes
}

func init() {
	var err error
	if dockerHost, err = getDockerHost(); err != nil {
		err = fmt.Errorf("failed to connect docker host: %v", err)
		common.PrintError(err)
	}

	var cli *client.Client
	cli, err = client.NewEnvClient()
	if err != nil {
		err = fmt.Errorf("failed to connect docker host: %v", err)
		return
	}
	defer cli.Close()

	cmd = &cobra.Command{
		Use:   os.Args[0],
		Short: "sebak composing network",
		Run: func(c *cobra.Command, args []string) {
			parseFlags()

			if flagForceClean {
				if err = cleanDocker(cli); err != nil {
					common.PrintError(err)
				}
			}

			// compose network
			nodes := composeNetwork()

			// Originally genesis block was not driven from `network id`. This
			// is just for testing network.
			genesisKeypair := keypair.Master(networkID).(*keypair.Full)
			log.Debug(
				"generate keypair for genesis block",
				"address", genesisKeypair.Address(),
				"secret seed", genesisKeypair.Seed(),
			)

			for _, node := range nodes {
				if err := runContainer(cli, genesisKeypair, node); err != nil {
					common.PrintError(err)
				}
			}

			// TODO gathering logs

			log.Info(fmt.Sprintf(
				"%d container was created and started to make sebak network",
				flagNumberOfNodes,
			))

			ctx := context.Background()
			var cl []types.Container
			if cl, err = cli.ContainerList(ctx, types.ContainerListOptions{All: true}); err != nil {
				log.Error("failed to get container list", "error", err)
				return
			}

			for _, c := range cl {
				var name string
				for _, n := range c.Names {
					if strings.HasPrefix(n[1:], dockerContainerNamePrefix) {
						name = n
						break
					}
				}
				if len(name) < 1 {
					continue
				}

				log.Debug(
					"container created",
					"name", name[1:],
					"id", c.ID[:3],
					"status", c.Status,
				)
			}

		},
	}

	cmd.Flags().UintVar(&flagNumberOfNodes, "n", 3, "number of node")
	cmd.Flags().StringVar(&flagImageName, "image", flagImageName, "docker image name for sebak")
	cmd.Flags().BoolVar(
		&flagForceClean,
		"force",
		flagForceClean,
		"remove the existing sebak containers",
	)
	cmd.Flags().StringVar(&flagLogLevel, "log-level", flagLogLevel, "log level, {crit, error, warn, info, debug}")
	cmd.Flags().StringVar(
		&flagSebakLogLevel,
		"sebak-log-level",
		flagSebakLogLevel,
		"sebak log level, {crit, error, warn, info, debug}",
	)
}

func main() {
	if err := cmd.Execute(); err != nil {
		common.PrintFlagsError(cmd, "", err)
	}
}
