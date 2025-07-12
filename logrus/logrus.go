package logrusld

import (
	"errors"
	"fmt"
	"net"
	"path"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/SandQuattro/logdoc-go-appender/common"
	"github.com/sirupsen/logrus"
)

const defaultAsyncBufferSize = 8192

var application string

var log = logrus.StandardLogger()

func GetLogger() *logrus.Logger {
	return log
}

func SetLogger(logger *logrus.Logger) {
	log = logger
}

var UnknownLogFormatError = errors.New("unknown log format")

const (
	JSON = iota
	TEXT
)

type Hook struct {
	sync.RWMutex
	conn                     net.Conn
	protocol                 string
	address                  string
	appName                  string
	alwaysSentFields         logrus.Fields
	hookOnlyPrefix           string
	TimeFormat               string
	fireChannel              chan *logrus.Entry
	AsyncBufferSize          int
	WaitUntilBufferFrees     bool
	Timeout                  time.Duration // Timeout for sending message.
	MaxSendRetries           int           // Declares how many times we will try to resend message.
	ReconnectBaseDelay       time.Duration // First reconnect delay.
	ReconnectDelayMultiplier float64       // Base multiplier for delay before reconnect.
	MaxReconnectRetries      int           // Declares how many times we will try to reconnect.
}

func (h *Hook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
		logrus.DebugLevel,
	}
}

// / Fire send message to logdoc.
// In async mode log message will be dropped if message buffer is full.
// If you want wait until message buffer frees – set WaitUntilBufferFrees to true.
func (h *Hook) Fire(entry *logrus.Entry) error {
	if h.fireChannel != nil { // Async mode.
		select {
		case h.fireChannel <- entry:
		default:
			if h.WaitUntilBufferFrees {
				h.fireChannel <- entry // Blocks the goroutine because buffer is full.
				return nil
			}
			// Drop message by default.
		}
		return nil
	}
	return h.sendMessage(entry)
}

func (h *Hook) sendMessage(entry *logrus.Entry) error {
	src := entry.Caller.Function + ":" + strconv.Itoa(entry.Caller.Line)

	// Используем общую функцию для обработки кастомных полей
	customFieldsProcessor := func(result *[]byte) {
		common.ProcessCustomFields(entry.Message, result)
	}

	result := common.BuildLogDocMessage(
		entry.Message,
		application,
		entry.Level.String(),
		src,
		h.conn,
		customFieldsProcessor,
	)

	_, err := h.conn.Write(result)
	if err != nil {
		logrus.Errorf("Ошибка записи в соединение, %s", err.Error())
	}
	return nil
}

func Init(proto string, address string, app string, format int) (net.Conn, error) {
	log.SetReportCaller(true)

	switch format {
	case JSON:
		log.Formatter = &logrus.JSONFormatter{
			CallerPrettyfier: func(f *runtime.Frame) (string, string) {
				filename := path.Base(f.File)
				return fmt.Sprintf("%s:%d", filename, f.Line), fmt.Sprintf("%s()", f.Function)
			},
			TimestampFormat: "02-01-2006 15:04:05.00000",
		}
	case TEXT:
		log.Formatter = &logrus.TextFormatter{
			ForceColors:     true,
			ForceQuote:      true,
			FullTimestamp:   true,
			TimestampFormat: "02.01.2006 15:04:05.000000",
			CallerPrettyfier: func(f *runtime.Frame) (string, string) {
				filename := path.Base(f.File)
				return fmt.Sprintf(" %s:%d", filename, f.Line), "" // fmt.Sprintf("%s()", f.Function)
			},
		}
	default:
		return nil, UnknownLogFormatError
	}

	log.SetLevel(logrus.DebugLevel)
	application = app

	hook, conn, err := NewHook(proto, address)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	log.AddHook(hook)
	return conn, nil
}

func NewHook(protocol, address string) (*Hook, net.Conn, error) {
	conn, err := net.Dial(protocol, address)
	if err != nil {
		logrus.Error("Error connecting LogDoc server, ", address, "; error:", err)
		return nil, nil, err
	}

	hook := &Hook{conn: conn, protocol: protocol, address: address}

	return hook, conn, nil
}

func (h *Hook) makeAsync() {
	if h.AsyncBufferSize == 0 {
		h.AsyncBufferSize = defaultAsyncBufferSize
	}
	h.fireChannel = make(chan *logrus.Entry, h.AsyncBufferSize)

	go func() {
		for entry := range h.fireChannel {
			if err := h.sendMessage(entry); err != nil {
				fmt.Println("Error during sending message to logdoc:", err)
			}
		}
	}()
}
