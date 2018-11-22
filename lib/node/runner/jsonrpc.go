package runner

import (
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/gorilla/rpc"
	jsonrpc "github.com/gorilla/rpc/json"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/storage"
)

const MaxLimitListOptions uint64 = 10000

type DBEchoArgs string
type DBEchoResult string

type DBHasArgs string
type DBHasResult bool

type DBGetArgs string
type DBGetResult storage.IterItem

type GetIteratorOptions struct {
	Reverse bool
	Cursor  []byte
	Limit   uint64
}

type DBGetIteratorArgs struct {
	Prefix  string
	Options GetIteratorOptions
}

type DBGetIteratorResult struct {
	Limit uint64
	Items []storage.IterItem
}

type jsonrpcDBApp struct {
	st *storage.LevelDBBackend
}

func (j *jsonrpcDBApp) Echo(r *http.Request, args *DBEchoArgs, result *DBEchoResult) error {
	*result = DBEchoResult(string(*args))
	return nil
}

func (j *jsonrpcDBApp) Has(r *http.Request, args *DBHasArgs, result *DBHasResult) error {
	o, err := j.st.Has(string(*args))
	if err != nil {
		return err
	}

	*result = DBHasResult(o)
	return nil
}

func (j *jsonrpcDBApp) Get(r *http.Request, args *DBGetArgs, result *DBGetResult) error {
	o, err := j.st.GetRaw(string(*args))
	if err != nil {
		return err
	}

	*result = DBGetResult{Key: []byte(*args), Value: o}
	return nil
}

func (j *jsonrpcDBApp) GetIterator(r *http.Request, args *DBGetIteratorArgs, result *DBGetIteratorResult) error {
	limit := args.Options.Limit
	if limit > MaxLimitListOptions {
		limit = MaxLimitListOptions
	}

	options := storage.NewDefaultListOptions(
		args.Options.Reverse,
		args.Options.Cursor,
		limit,
	)

	it, closeFunc := j.st.GetIterator(args.Prefix, options)
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
}

func newJSONRPCServer(endpoint *common.Endpoint, st *storage.LevelDBBackend) *jsonrpcServer {
	return &jsonrpcServer{
		endpoint: endpoint,
		st:       st,
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

	dbApp := &jsonrpcDBApp{st: j.st}
	s.RegisterService(dbApp, "DB")

	router := mux.NewRouter()

	path := j.endpoint.Path
	if len(path) < 1 {
		path = "/"
	}
	router.Handle(path, s)

	return router
}

func (j *jsonrpcServer) Start() error {
	router := j.Ready()

	err := func() error {
		if strings.ToLower(j.endpoint.Scheme) == "http" {
			return http.ListenAndServe(j.endpoint.Host, router)
		}

		tlsCertFile := j.endpoint.Query().Get("TLSCertFile")
		tlsKeyFile := j.endpoint.Query().Get("TLSKeyFile")

		return http.ListenAndServeTLS(j.endpoint.Host, tlsCertFile, tlsKeyFile, router)
	}()

	if err == http.ErrServerClosed {
		return nil
	}

	return err
}
