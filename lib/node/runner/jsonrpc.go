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

var MaxLimitListOptions uint64 = storage.DefaultMaxLimitListOptions * 10

type EchoArgs string
type EchoResult string

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

type JSONRPCMainApp struct {
}

func (j *JSONRPCMainApp) Echo(r *http.Request, args *EchoArgs, result *EchoResult) error {
	*result = EchoResult(string(*args))
	return nil
}

type JSONRPCDBApp struct {
	st *storage.LevelDBBackend
}

func (j *JSONRPCDBApp) Has(r *http.Request, args *DBHasArgs, result *DBHasResult) error {
	o, err := j.st.Has(string(*args))
	if err != nil {
		return err
	}

	*result = DBHasResult(o)
	return nil
}

func (j *JSONRPCDBApp) Get(r *http.Request, args *DBGetArgs, result *DBGetResult) error {
	o, err := j.st.GetRaw(string(*args))
	if err != nil {
		return err
	}

	*result = DBGetResult{Key: []byte(*args), Value: o}
	return nil
}

func (j *JSONRPCDBApp) GetIterator(r *http.Request, args *DBGetIteratorArgs, result *DBGetIteratorResult) error {
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

type JSONRPCServer struct {
	endpoint *common.Endpoint
	st       *storage.LevelDBBackend
}

func NewJSONRPCServer(endpoint *common.Endpoint, st *storage.LevelDBBackend) *JSONRPCServer {
	return &JSONRPCServer{
		endpoint: endpoint,
		st:       st,
	}
}

type Server struct {
	*rpc.Server
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

func (j *JSONRPCServer) Ready() *mux.Router {
	s := &Server{Server: rpc.NewServer()}
	s.RegisterCodec(jsonrpc.NewCodec(), "application/json")
	s.RegisterCodec(jsonrpc.NewCodec(), "application/json;charset=UTF-8")

	mainApp := &JSONRPCMainApp{}
	s.RegisterService(mainApp, "Main")

	dbApp := &JSONRPCDBApp{st: j.st}
	s.RegisterService(dbApp, "DB")

	router := mux.NewRouter()

	path := j.endpoint.Path
	if len(path) < 1 {
		path = "/"
	}
	router.Handle(path, s)

	return router
}

func (j *JSONRPCServer) Start() error {
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
