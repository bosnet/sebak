package sebakstorage

import (
	"errors"
	"net/url"

	"boscoin.io/sebak/lib/common"
)

var SupportedStorageType []string = []string{
	"memory",
	"file",
}

type IterItem struct {
	N     int64
	Key   []byte
	Value []byte
}

type Item struct {
	Key   string
	Value interface{}
}

type Model struct {
}

func NewStorage(config *Config) (st *LevelDBBackend, err error) {
	st = &LevelDBBackend{}
	if err = st.Init(config); err != nil {
		return
	}

	return
}

type Config url.URL

func NewConfigFromURL(u *url.URL) *Config {
	return (*Config)(u)
}

func NewConfigFromString(s string) (e *Config, err error) {
	var u *url.URL
	if u, err = url.Parse(s); err != nil {
		return
	}

	if _, found := common.InStringArray(SupportedStorageType, u.Scheme); !found {
		err = errors.New("unsupported storage type")
		return
	}

	e = NewConfigFromURL(u)
	return
}

func (e *Config) String() string {
	return (&url.URL{
		Scheme: e.Scheme,
		Host:   e.Host,
		Path:   e.Path,
	}).String()
}

func (e *Config) Query() url.Values {
	return (*url.URL)(e).Query()
}

func (e *Config) UnmarshalJSON(b []byte) error {
	p, err := ParseConfig(string(b)[1 : len(string(b))-1])
	if err != nil {
		return err
	}

	*e = *p

	return nil
}

func ParseConfig(s string) (u *Config, err error) {
	var parsed *url.URL
	parsed, err = url.Parse(s)
	if err != nil {
		return
	}
	if len(parsed.Scheme) < 1 {
		err = errors.New("missing scheme")
		return
	}

	u = (*Config)(parsed)

	return
}
