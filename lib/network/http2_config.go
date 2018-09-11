package network

import (
	"errors"
	"strings"
	"time"

	"boscoin.io/sebak/lib/common"
)

type HTTP2NetworkConfig struct {
	NodeName string
	Endpoint *common.Endpoint
	Addr     string

	ReadTimeout,
	ReadHeaderTimeout,
	WriteTimeout,
	IdleTimeout time.Duration

	TLSCertFile,
	TLSKeyFile string
}

func NewHTTP2NetworkConfigFromEndpoint(nodeName string, endpoint *common.Endpoint) (config *HTTP2NetworkConfig, err error) {
	query := endpoint.Query()

	var ReadTimeout time.Duration = 0
	var ReadHeaderTimeout time.Duration = 0
	var WriteTimeout time.Duration = 0
	var IdleTimeout time.Duration = 5
	var TLSCertFile, TLSKeyFile string

	if ReadTimeout, err = time.ParseDuration(common.GetUrlQuery(query, "ReadTimeout", "0s")); err != nil {
		return
	}
	if ReadTimeout < 0*time.Second {
		err = errors.New("invalid 'ReadTimeout'")
		return
	}

	if ReadHeaderTimeout, err = time.ParseDuration(common.GetUrlQuery(query, "ReadHeaderTimeout", "0s")); err != nil {
		return
	}
	if ReadHeaderTimeout < 0*time.Second {
		err = errors.New("invalid 'ReadHeaderTimeout'")
		return
	}

	if WriteTimeout, err = time.ParseDuration(common.GetUrlQuery(query, "WriteTimeout", "0s")); err != nil {
		return
	}
	if WriteTimeout < 0*time.Second {
		err = errors.New("invalid 'WriteTimeout'")
		return
	}

	if IdleTimeout, err = time.ParseDuration(common.GetUrlQuery(query, "IdleTimeout", "0s")); err != nil {
		return
	}
	if IdleTimeout < 0*time.Second {
		err = errors.New("invalid 'IdleTimeout'")
		return
	}

	TLSCertFile = query.Get("TLSCertFile")
	TLSKeyFile = query.Get("TLSKeyFile")

	if strings.ToLower(endpoint.Scheme) == "https" && (len(TLSCertFile) < 1 || len(TLSKeyFile) < 1) {
		err = errors.New("HTTPS needs `TLSCertFile` and `TLSKeyFile`")
		return
	}

	config = &HTTP2NetworkConfig{
		NodeName:          nodeName,
		Endpoint:          endpoint,
		Addr:              endpoint.Host,
		ReadTimeout:       ReadTimeout,
		ReadHeaderTimeout: ReadHeaderTimeout,
		WriteTimeout:      WriteTimeout,
		IdleTimeout:       IdleTimeout,
		TLSCertFile:       TLSCertFile,
		TLSKeyFile:        TLSKeyFile,
	}

	return
}

func (config HTTP2NetworkConfig) IsHTTPS() bool {
	return len(config.TLSCertFile) > 0 && len(config.TLSKeyFile) > 0
}

func (config HTTP2NetworkConfig) String() string {
	return string(common.MustJSONMarshal(config))
}
