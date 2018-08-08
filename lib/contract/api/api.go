package api

type API interface {
	Helloworld(string) (string, error)
}
