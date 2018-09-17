package runner

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"boscoin.io/sebak/lib/storage"
)

type GetBlocksOptionsMode string

const (
	GetBlocksOptionsModeHeader GetBlocksOptionsMode = "header" // default
	GetBlocksOptionsModeBlock  GetBlocksOptionsMode = "block"
	GetBlocksOptionsModeFull   GetBlocksOptionsMode = "full"
)

type GetBlocksOptions struct {
	*storage.DefaultListOptions

	r *http.Request

	HeightRange [2]uint64
	Hashes      []string
	Mode        GetBlocksOptionsMode
}

func NewGetBlocksOptionsFromRequest(r *http.Request) (options *GetBlocksOptions, err error) {
	options = &GetBlocksOptions{r: r}

	if r == nil {
		options.DefaultListOptions = storage.NewDefaultListOptions(
			false,
			nil,
			storage.DefaultMaxLimitListOptions,
		)
	} else {
		if options.DefaultListOptions, err = storage.NewDefaultListOptionsFromQuery(r.URL.Query()); err != nil {
			return
		}

		if err = options.parseBlockHeightRange(); err != nil {
			return
		}
		if err = options.parseBlockHashes(); err != nil {
			return
		}
		if err = options.parseGetBlocksOptionsMode(); err != nil {
			return
		}
	}

	return
}

func (g *GetBlocksOptions) parseBlockHeightRange() (err error) {
	queries := g.r.URL.Query()
	heightRangeValue := queries.Get("height-range")
	if len(heightRangeValue) < 1 {
		return
	}

	sp := strings.SplitN(heightRangeValue, "-", 2)
	if len(sp) != 2 {
		err = fmt.Errorf("invalid `height-range` format")
		return
	}

	var start, end uint64
	if start, err = strconv.ParseUint(sp[0], 10, 64); err != nil {
		err = fmt.Errorf("`height-range` value is not uint64: %v", err)
		return
	}
	if end, err = strconv.ParseUint(sp[1], 10, 64); err != nil {
		err = fmt.Errorf("`height-range` value is not uint64: %v", err)
		return
	}
	if end <= start {
		err = fmt.Errorf("invalid `height-range` value: start must be bigger than end")
		return
	}

	g.HeightRange[0] = start
	g.HeightRange[1] = end

	return
}

func (g GetBlocksOptions) Height() uint64 {
	return g.HeightRange[1] - g.HeightRange[0]
}

func (g *GetBlocksOptions) parseBlockHashes() (err error) {
	queries := g.r.URL.Query()

	// hashes
	var hashesPost []string
	hashesGet := queries["hash"]

	// `hash` can get from post data
	if g.r.Method == "POST" {
		if g.r.Header.Get("Content-Type") != "application/json" {
			err = fmt.Errorf("`Content-Type` must be 'application/json'")
			return
		}

		var body []byte
		if body, err = ioutil.ReadAll(g.r.Body); err != nil {
			return
		} else if len(strings.TrimSpace(string(body))) < 1 {
			goto end
		}

		if err = json.Unmarshal(body, &hashesPost); err != nil {
			return
		}
	}

end:
	hashMap := map[string]bool{}
	for _, hash := range append(hashesGet, hashesPost...) {
		if len(hash) < 1 {
			continue
		}

		if _, ok := hashMap[hash]; ok {
			continue
		}
		hashMap[hash] = true
		g.Hashes = append(g.Hashes, hash)
	}

	return
}

func (g *GetBlocksOptions) parseGetBlocksOptionsMode() error {
	s := g.r.URL.Query().Get("mode")
	if len(s) < 1 {
		g.Mode = GetBlocksOptionsModeHeader
		return nil
	}

	var mode GetBlocksOptionsMode
	switch GetBlocksOptionsMode(s) {
	case GetBlocksOptionsModeBlock:
		mode = GetBlocksOptionsModeBlock
	case GetBlocksOptionsModeHeader:
		mode = GetBlocksOptionsModeHeader
	case GetBlocksOptionsModeFull:
		mode = GetBlocksOptionsModeFull
	default:
		return fmt.Errorf("unknown `mode`")
	}

	g.Mode = mode

	return nil
}

func (g GetBlocksOptions) Template() string {
	return "{?cursor,limit,order,height-range,hash,mode}"
}

func (g GetBlocksOptions) URLValues() url.Values {
	v := g.DefaultListOptions.URLValues()
	if g.Height() > 0 {
		v.Set("height-range", fmt.Sprintf("%d-%d", g.HeightRange[0], g.HeightRange[1]))
	}
	if len(g.Hashes) > 0 {
		v["hash"] = g.Hashes
	}
	v.Set("mode", string(g.Mode))

	return v
}

func (g *GetBlocksOptions) SetHeightRange(h [2]uint64) *GetBlocksOptions {
	g.HeightRange = h
	return g
}

func (g *GetBlocksOptions) SetHashes(h []string) *GetBlocksOptions {
	g.Hashes = h
	return g
}

func (g *GetBlocksOptions) SetMode(mode GetBlocksOptionsMode) *GetBlocksOptions {
	g.Mode = mode
	return g
}
