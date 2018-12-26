package runner

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/rpc"
	jsonrpc "github.com/gorilla/rpc/json"
	"golang.org/x/sync/syncmap"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/errors"
	"boscoin.io/sebak/lib/storage"
)

const MaxLimitListOptions uint64 = 10000
const MaxSnapshots uint64 = 500

type DBEchoArgs string
type DBEchoResult string

type DBOpenSnapshot struct{}

type DBOpenSnapshotResult struct {
	Snapshot string `json:"snapshot"`
}

type DBReleaseSnapshot struct {
	Snapshot string `json:"snapshot"`
}

type DBReleaseSnapshotResult bool

type DBHasArgs struct {
	Snapshot string `json:"snapshot"`
	Key      string `json:"key"`
}

type DBHasResult bool

type DBGetArgs struct {
	Snapshot string `json:"snapshot"`
	Key      string `json:"key"`
}

type DBGetResult storage.IterItem

type GetIteratorOptions struct {
	Reverse bool   `json:"reverse"`
	Cursor  []byte `json:"cursor"`
	Limit   uint64 `json:"limit"`
}

type DBGetIteratorArgs struct {
	Snapshot string             `json:"snapshot"`
	Prefix   string             `json:"prefix"`
	Options  GetIteratorOptions `json:"options"`
}

type DBGetIteratorResult struct {
	Limit uint64             `json:"limit"`
	Items []storage.IterItem `json:"items"`
}

type jsonrpcDBApp struct {
	st        *storage.LevelDBBackend
	snapshots *expireSnapshots
}

type expireSnapshots struct {
	sync.RWMutex
	st           *storage.LevelDBBackend
	interval     time.Duration
	maxSnapshots uint64
	ticker       *time.Ticker
	snapshots    *syncmap.Map
	expires      *syncmap.Map
}

func newExpireSnapshots(st *storage.LevelDBBackend, interval time.Duration, maxSnapshots uint64) *expireSnapshots {
	return &expireSnapshots{
		st:           st,
		interval:     interval,
		maxSnapshots: maxSnapshots,
		ticker:       time.NewTicker(time.Second * 10),
		snapshots:    &syncmap.Map{},
		expires:      &syncmap.Map{},
	}
}

func (j *expireSnapshots) len() (l int) {
	j.snapshots.Range(func(_, _ interface{}) bool {
		l++
		return true
	})

	return
}

func (j *expireSnapshots) newSnapshot() (string, *storage.LevelDBBackend, error) {
	if j.len() >= int(j.maxSnapshots) {
		return "", nil, errors.SnapshotLimitReached
	}

	j.Lock()
	defer j.Unlock()

	st, err := j.st.OpenSnapshot()
	if err != nil {
		return "", nil, err
	}

	key := common.GetUniqueIDFromUUID()
	j.snapshots.Store(key, st)
	j.updateExpire(key)

	log.Debug("new snapshot created", "key", key, "current", j.len())
	return key, st, nil
}

func (j *expireSnapshots) snapshot(key string) (*storage.LevelDBBackend, bool) {
	j.RLock()
	defer j.RUnlock()

	s, ok := j.snapshots.Load(key)
	if !ok {
		return nil, false
	}

	j.updateExpire(key)
	return s.(*storage.LevelDBBackend), true
}

func (j *expireSnapshots) expire(key string) bool {
	go func() {
		log.Debug("snapshot expired", "key", key, "current", j.len())
	}()
	return j.release_(key)
}

func (j *expireSnapshots) release(key string) bool {
	go func() {
		log.Debug("snapshot released", "key", key, "current", j.len())
	}()
	return j.release_(key)
}

func (j *expireSnapshots) release_(key string) bool {
	st, ok := j.snapshot(key)
	if !ok {
		return false
	}

	j.Lock()
	defer j.Unlock()

	if st.Core != nil {
		st.Core.(*storage.Snapshot).Release()
	}
	j.snapshots.Delete(key)
	j.expires.Delete(key)

	return true
}

func (j *expireSnapshots) isExpired(key string) bool {
	j.RLock()
	defer j.RUnlock()

	t, ok := j.expires.Load(key)
	if !ok {
		return true
	}

	return time.Now().Sub(t.(time.Time)) > time.Second
}

func (j *expireSnapshots) updateExpire(key string) {
	j.expires.Store(key, time.Now().Add(j.interval))
}

func (j *expireSnapshots) start() {
	go func() {
		for _ = range j.ticker.C {
			var expired []string
			j.snapshots.Range(func(k, _ interface{}) bool {
				key := k.(string)
				if j.isExpired(key) {
					expired = append(expired, key)
				}
				return true
			})

			for _, key := range expired {
				j.expire(key)
			}
		}
	}()
}

func (j *expireSnapshots) stop() {
	j.ticker.Stop()
}

func newJSONRPCDBApp(st *storage.LevelDBBackend) *jsonrpcDBApp {
	app := &jsonrpcDBApp{
		st:        st,
		snapshots: newExpireSnapshots(st, time.Minute*1, MaxSnapshots),
	}

	return app
}

func (j *jsonrpcDBApp) stop() {
	j.snapshots.stop()
}

func (j *jsonrpcDBApp) Echo(r *http.Request, args *DBEchoArgs, result *DBEchoResult) error {
	*result = DBEchoResult(string(*args))
	return nil
}

func (j *jsonrpcDBApp) OpenSnapshot(r *http.Request, args *DBOpenSnapshot, result *DBOpenSnapshotResult) error {
	key, _, err := j.snapshots.newSnapshot()
	if err != nil {
		return err
	}
	*result = DBOpenSnapshotResult{Snapshot: key}
	return nil
}

func (j *jsonrpcDBApp) ReleaseSnapshot(r *http.Request, args *DBReleaseSnapshot, result *DBReleaseSnapshotResult) error {
	ok := j.snapshots.release(args.Snapshot)
	*result = DBReleaseSnapshotResult(ok)
	return nil
}

func (j *jsonrpcDBApp) Has(r *http.Request, args *DBHasArgs, result *DBHasResult) error {
	if len(args.Snapshot) < 1 {
		return fmt.Errorf("snapshot must be given")
	}

	st, found := j.snapshots.snapshot(args.Snapshot)
	if !found {
		return errors.SnapshotNotFound
	}

	o, err := st.Has(args.Key)
	if err != nil {
		return err
	}

	*result = DBHasResult(o)
	return nil
}

func (j *jsonrpcDBApp) Get(r *http.Request, args *DBGetArgs, result *DBGetResult) error {
	if len(args.Snapshot) < 1 {
		return fmt.Errorf("snapshot must be given")
	}

	st, found := j.snapshots.snapshot(args.Snapshot)
	if !found {
		return errors.SnapshotNotFound
	}

	o, err := st.GetRaw(args.Key)
	if err != nil {
		return err
	}

	*result = DBGetResult{Key: []byte(args.Key), Value: o}
	return nil
}

func (j *jsonrpcDBApp) GetIterator(r *http.Request, args *DBGetIteratorArgs, result *DBGetIteratorResult) error {
	if len(args.Snapshot) < 1 {
		return fmt.Errorf("snapshot must be given")
	}

	st, found := j.snapshots.snapshot(args.Snapshot)
	if !found {
		return errors.SnapshotNotFound
	}

	limit := args.Options.Limit
	if limit > MaxLimitListOptions {
		limit = MaxLimitListOptions
	}

	options := storage.NewDefaultListOptions(
		args.Options.Reverse,
		args.Options.Cursor,
		limit,
	)

	it, closeFunc := st.GetIterator(args.Prefix, options)
	defer closeFunc()

	collected := []storage.IterItem{}
	for {
		v, hasNext := it()
		if !hasNext {
			break
		}

		collected = append(collected, v.Clone())
	}

	result.Items = collected
	result.Limit = limit

	return nil
}

type jsonrpcServer struct {
	endpoint *common.Endpoint
	st       *storage.LevelDBBackend
	server   *http.Server
	app      *jsonrpcDBApp
}

func newJSONRPCServer(endpoint *common.Endpoint, st *storage.LevelDBBackend) *jsonrpcServer {
	return &jsonrpcServer{
		endpoint: endpoint,
		st:       st,
		server:   &http.Server{Addr: endpoint.Host},
	}
}

type jsonrpcInternalServer struct {
	*rpc.Server
}

func (s *jsonrpcInternalServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set(
		"Access-Control-Allow-Headers",
		"Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization",
	)

	if r.Method == "OPTIONS" {
		return
	}

	s.Server.ServeHTTP(w, r)
}

func (j *jsonrpcServer) Ready() *mux.Router {
	s := &jsonrpcInternalServer{Server: rpc.NewServer()}
	s.RegisterCodec(jsonrpc.NewCodec(), "application/json")
	s.RegisterCodec(jsonrpc.NewCodec(), "application/json;charset=UTF-8")

	j.app = newJSONRPCDBApp(j.st)
	s.RegisterService(j.app, "DB")

	router := mux.NewRouter()

	path := j.endpoint.Path
	if len(path) < 1 {
		path = "/"
	}
	router.Handle(path, s)

	return router
}

func (j *jsonrpcServer) Start() error {
	j.server.Handler = j.Ready()
	j.app.snapshots.start()

	err := func() error {
		if strings.ToLower(j.endpoint.Scheme) == "http" {
			return j.server.ListenAndServe()
		}

		tlsCertFile := j.endpoint.Query().Get("TLSCertFile")
		tlsKeyFile := j.endpoint.Query().Get("TLSKeyFile")

		return j.server.ListenAndServeTLS(tlsCertFile, tlsKeyFile)
	}()

	if err == http.ErrServerClosed {
		return nil
	}

	return err
}

func (j *jsonrpcServer) Stop() {
	if j.app != nil {
		j.app.stop()
	}

	j.server.Close()
}
