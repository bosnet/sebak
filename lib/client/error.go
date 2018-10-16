package client

type Error struct {
	Problem Problem
}

func (e Error) Error() string {
	return e.Problem.Title
}
