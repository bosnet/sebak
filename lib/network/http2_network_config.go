package sebaknetwork

import (
	"errors"
	"io"
	"os"
	"time"

	"boscoin.io/sebak/lib/common"
)

type TLSInfo interface {
	GetTLSCertInfo(*sebakcommon.Endpoint) (string, error)
	GetTLSKeyInfo(*sebakcommon.Endpoint) (string, error)
}

type ValidTLSInfo struct{}

func (t ValidTLSInfo) GetTLSCertInfo(endpoint *sebakcommon.Endpoint) (result string, err error) {
	query := endpoint.Query()
	if v := query.Get("TLSCertFile"); len(v) < 1 {
		err = errors.New("'TLSCertFile' is missing")
		return
	} else {
		result = v
	}
	return
}

func (t ValidTLSInfo) GetTLSKeyInfo(endpoint *sebakcommon.Endpoint) (result string, err error) {
	query := endpoint.Query()
	if v := query.Get("TLSKeyFile"); len(v) < 1 {
		err = errors.New("'TLSKeyFile' is missing")
		return
	} else {
		result = v
	}
	return
}

type HTTP2NetworkConfig struct {
	NodeName string
	Addr     string

	ReadTimeout,
	ReadHeaderTimeout,
	WriteTimeout,
	IdleTimeout time.Duration

	TLSCertFile,
	TLSKeyFile string

	HTTP2LogOutput io.Writer
}

func NewHTTP2NetworkConfigFromEndpoint(tls TLSInfo, endpoint *sebakcommon.Endpoint) (config HTTP2NetworkConfig, err error) {
	query := endpoint.Query()

	var NodeName string
	var ReadTimeout time.Duration = 0
	var ReadHeaderTimeout time.Duration = 0
	var WriteTimeout time.Duration = 0
	var IdleTimeout time.Duration = 5
	var TLSCertFile, TLSKeyFile string
	var HTTP2LogOutput io.Writer

	if ReadTimeout, err = time.ParseDuration(sebakcommon.GetUrlQuery(query, "ReadTimeout", "0s")); err != nil {
		return
	}
	if ReadTimeout < 0*time.Second {
		err = errors.New("invalid 'ReadTimeout'")
		return
	}

	if ReadHeaderTimeout, err = time.ParseDuration(sebakcommon.GetUrlQuery(query, "ReadHeaderTimeout", "0s")); err != nil {
		return
	}
	if ReadHeaderTimeout < 0*time.Second {
		err = errors.New("invalid 'ReadHeaderTimeout'")
		return
	}

	if WriteTimeout, err = time.ParseDuration(sebakcommon.GetUrlQuery(query, "WriteTimeout", "0s")); err != nil {
		return
	}
	if WriteTimeout < 0*time.Second {
		err = errors.New("invalid 'WriteTimeout'")
		return
	}

	if IdleTimeout, err = time.ParseDuration(sebakcommon.GetUrlQuery(query, "IdleTimeout", "0s")); err != nil {
		return
	}
	if IdleTimeout < 0*time.Second {
		err = errors.New("invalid 'IdleTimeout'")
		return
	}

	TLSCertFile, err = tls.GetTLSCertInfo(endpoint)
	if err != nil {
		return
	}

	TLSKeyFile, err = tls.GetTLSKeyInfo(endpoint)
	if err != nil {
		return
	}

	if v := query.Get("NodeName"); len(v) < 1 {
		err = errors.New("`NodeName` must be given")
		return
	} else {
		NodeName = v
	}

	if v := query.Get("HTTP2LogOutput"); len(v) < 1 {
		HTTP2LogOutput = os.Stdout
	} else {
		HTTP2LogOutput, err = os.OpenFile(v, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return
		}
	}

	config = HTTP2NetworkConfig{
		NodeName:          NodeName,
		Addr:              endpoint.Host,
		ReadTimeout:       ReadTimeout,
		ReadHeaderTimeout: ReadHeaderTimeout,
		WriteTimeout:      WriteTimeout,
		IdleTimeout:       IdleTimeout,
		TLSCertFile:       TLSCertFile,
		TLSKeyFile:        TLSKeyFile,
		HTTP2LogOutput:    HTTP2LogOutput,
	}

	return
}
