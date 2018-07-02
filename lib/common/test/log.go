package test

import (
	"os"

	logging "github.com/inconshreveable/log15"
)

func LogHandler() logging.Handler {
	handlers := map[string]func() logging.Handler{
		"null": func() logging.Handler {
			return logging.DiscardHandler()
		},
		"stdout": func() logging.Handler {
			return logging.CallerStackHandler("%+v", logging.StdoutHandler)
		},
	}

	handler := handlers["stdout"]
	if h, ok := handlers[os.Getenv("SEBAK_LOG_HANDLER")]; ok {
		handler = h
	}

	return handler()
}
