package sync

import (
	"context"
	"errors"
	"net/http"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node"
)

type mockConnectionManager struct {
	allConnected  []string
	allValidators []string
	getNodeFunc   func(addr string) node.Node
}

func (m *mockConnectionManager) GetNodeAddress() string {
	return ""
}

func (m *mockConnectionManager) Broadcast(common.Message) {}
func (m *mockConnectionManager) Start()                   {}

func (m *mockConnectionManager) GetConnection(string) network.NetworkClient {
	return nil
}

func (m *mockConnectionManager) AllConnected() []string {
	return m.allConnected
}

func (m *mockConnectionManager) AllValidators() []string {
	return m.allValidators
}

func (m *mockConnectionManager) CountConnected() int {
	return len(m.allConnected)
}

func (m *mockConnectionManager) IsReady() bool {
	return true
}

func (m *mockConnectionManager) Discovery(network.DiscoveryMessage) error {
	return nil
}

type mockDoer struct {
	handleFunc func(*http.Request) (*http.Response, error)
}

func (d mockDoer) Do(req *http.Request) (*http.Response, error) {
	if d.handleFunc == nil {
		return nil, errors.New("not implemented")
	}
	return d.handleFunc(req)
}

type mockFetcher struct {
	fetchFunc func(context.Context, *SyncInfo) (*SyncInfo, error)
}

func (f mockFetcher) Fetch(ctx context.Context, si *SyncInfo) (*SyncInfo, error) {
	return f.fetchFunc(ctx, si)
}

type mockValidator struct {
	validateFunc func(context.Context, *SyncInfo) error
}

func (v mockValidator) Validate(ctx context.Context, si *SyncInfo) error {
	return v.validateFunc(ctx, si)
}
