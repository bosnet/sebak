package common

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"time"

	logging "github.com/inconshreveable/log15"

	"boscoin.io/sebak/lib/errors"
)

var (
	DefaultLogLevel   logging.Lvl     = logging.LvlInfo
	DefaultLogHandler logging.Handler = logging.StreamHandler(os.Stdout, logging.TerminalFormat())
)

// `formatJSONValue` and `JsonFormatEx` was derived from
// https://github.com/inconshreveable/log15/blob/199fca55789248e0520a3bd33e9045799738e793/format.go#L131
// .
const errorKey = "LOG15_ERROR"

func formatJSONValue(value interface{}) (result interface{}) {
	defer func() {
		if err := recover(); err != nil {
			if v := reflect.ValueOf(value); v.Kind() == reflect.Ptr && v.IsNil() {
				result = "nil"
			} else {
				panic(err)
			}
		}
	}()

	switch v := value.(type) {
	case json.Marshaler, *errors.Error:
		return v
	case time.Time:
		return FormatISO8601(v)
	case error:
		return v.Error()
	case fmt.Stringer:
		return v.String()
	default:
		return v
	}
}

func JsonFormatEx(pretty, lineSeparated bool) logging.Format {
	jsonMarshal := json.Marshal
	if pretty {
		jsonMarshal = func(v interface{}) ([]byte, error) {
			return JSONMarshalIndent(v)
		}
	}

	return logging.FormatFunc(func(r *logging.Record) []byte {
		props := make(map[string]interface{})

		props[r.KeyNames.Time] = r.Time
		props[r.KeyNames.Lvl] = r.Lvl.String()
		props[r.KeyNames.Msg] = r.Msg

		for i := 0; i < len(r.Ctx); i += 2 {
			k, ok := r.Ctx[i].(string)
			if !ok {
				props[errorKey] = fmt.Sprintf("%+v is not a string key", r.Ctx[i])
			}
			props[k] = formatJSONValue(r.Ctx[i+1])
		}

		b, err := jsonMarshal(props)
		if err != nil {
			b, _ = jsonMarshal(map[string]string{
				errorKey: err.Error(),
			})
			return b
		}

		if lineSeparated {
			b = append(b, '\n')
		}

		return b
	})
}

// NopLogger returns a Logger with a no-op (nil) Logger
func NopLogger() logging.Logger {
	return &nopLogger{}
}

type nopLogger struct{}

func (l *nopLogger) New(ctx ...interface{}) logging.Logger {
	return l
}

func (l nopLogger) GetHandler() logging.Handler {
	return nil
}

func (l nopLogger) SetHandler(logging.Handler)           {}
func (l nopLogger) Debug(msg string, ctx ...interface{}) {}
func (l nopLogger) Info(msg string, ctx ...interface{})  {}
func (l nopLogger) Warn(msg string, ctx ...interface{})  {}
func (l nopLogger) Error(msg string, ctx ...interface{}) {}
func (l nopLogger) Crit(msg string, ctx ...interface{})  {}
