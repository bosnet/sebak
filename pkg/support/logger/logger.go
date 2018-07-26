package logger

import (
	"bytes"
	"fmt"
	"github.com/go-kit/kit/log"
	"github.com/go-logfmt/logfmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	Default = iota
	Red
	Yellow
	White
	Black
)

// https://en.wikipedia.org/wiki/ANSI_escape_code#Colors
var (
	resetColorBytes = []byte("\x1b[39;49;22m")

	fgColorBytes = [][]byte{
		[]byte("\x1b[39m"),
		[]byte(fmt.Sprintf("\x1b[%dm", 91)),
		[]byte(fmt.Sprintf("\x1b[%dm", 93)),
		[]byte(fmt.Sprintf("\x1b[%dm", 97)),
		[]byte(fmt.Sprintf("\x1b[%dm", 90)),
	}
	bgColorBytes = [][]byte{
		[]byte("\x1b[49m"),
		[]byte(fmt.Sprintf("\x1b[%dm", 101)),
		[]byte(fmt.Sprintf("\x1b[%dm", 103)),
		[]byte(fmt.Sprintf("\x1b[%dm", 107)),
		[]byte(fmt.Sprintf("\x1b[%dm", 100)),
	}
)

type Logger struct {
	logger log.Logger
}

func NewLogger(module string) *Logger {
	var logger log.Logger

	format := log.TimestampFormat(func() time.Time { return time.Now().UTC() }, time.RFC3339)

	logger = NewCustomLogger(log.NewSyncWriter(os.Stdout))
	logger = log.With(logger, "module", module)
	logger = log.With(logger, "ts", format, "caller", log.Caller(4))

	return &Logger{
		logger: logger,
	}
}

func (o *Logger) Info(keyvals ...interface{}) {
	o.logger.Log(append(keyvals, "level", "INF")...)
}

func (o *Logger) Debug(keyvals ...interface{}) {
	o.logger.Log(append(keyvals, "level", "DBG")...)
}

func (o *Logger) Warn(keyvals ...interface{}) {
	o.logger.Log(append(keyvals, "level", "WRN")...)
}

type customLogger struct {
	io.Writer
}

type logfmtEncoder struct {
	*logfmt.Encoder
	buf bytes.Buffer
}

func (l *logfmtEncoder) Reset() {
	l.Encoder.Reset()
	l.buf.Reset()
}

var encoderPool = sync.Pool{
	New: func() interface{} {
		var enc logfmtEncoder
		enc.Encoder = logfmt.NewEncoder(&enc.buf)
		return &enc
	},
}

func NewCustomLogger(w io.Writer) log.Logger {
	return &customLogger{w}
}

func (l *customLogger) Log(keyvals ...interface{}) error {
	enc := encoderPool.Get().(*logfmtEncoder)
	enc.Reset()
	defer encoderPool.Put(enc)

	var msg string
	var module string
	var level string
	var ts string
	var caller string

	for i := 0; i < len(keyvals)-1; i += 2 {
		switch keyvals[i] {
		case "level":
			level = keyvals[i+1].(string)
		case "module":
			module = keyvals[i+1].(string)
		case "ts":
			if v, ok := keyvals[i+1].(fmt.Stringer); ok {
				ts = v.String()
			}
		case "msg":
			msg = keyvals[i+1].(string)
		case "caller":
			if v, ok := keyvals[i+1].(fmt.Stringer); ok {
				caller = v.String()
			}
		}
	}

	switch level {
	case "DBG":
		enc.buf.Write(fgColorBytes[Black])
	case "WRN":
		enc.buf.Write(fgColorBytes[Red])
	}

	enc.buf.WriteString(fmt.Sprintf("%s [%s] %7s: %-40s ", ts, level, strings.ToUpper(module), msg))

	for i := 0; i < len(keyvals)-1; i += 2 {
		switch keyvals[i] {
		case "level", "module", "ts", "msg", "caller":
		default:
			enc.buf.WriteString(fmt.Sprintf("%s%v ", key(keyvals[i]), keyvals[i+1]))
		}
	}

	enc.buf.WriteString(fmt.Sprintf("%s%s", key("caller"), caller))
	enc.buf.Write(resetColorBytes)

	if err := enc.EndRecord(); err != nil {
		return err
	}

	if _, err := l.Write(enc.buf.Bytes()); err != nil {
		return err
	}

	return nil
}

func key(name interface{}) string {
	return fmt.Sprintf("%s%s=%s", fgColorBytes[Black], name, fgColorBytes[Default])
}
