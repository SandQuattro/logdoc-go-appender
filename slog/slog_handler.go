package slogld

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/SandQuattro/logdoc-go-appender/common"
)

var log = slog.Default()

// LogdocHandler is a Handler that writes log records to the Logdoc.
type LogdocHandler struct {
	slog.Handler
	levels      []slog.Level
	proto       string
	address     string
	application string
	Connection  net.Conn
}

// NewLogdocHandler creates a LogdocHandler that writes to w,
// using the given options.
func NewLogdocHandler(
	handler slog.Handler,
	levels []slog.Level,
	proto, address, application string,
) *LogdocHandler {

	conn, err := networkWriter(proto, address)
	if err != nil {
		log.Error("Ошибка соединения с LogDoc сервером")
		return nil
	}

	return &LogdocHandler{
		Handler:     handler,
		levels:      levels,
		proto:       proto,
		address:     address,
		application: application,
		Connection:  conn,
	}
}

func GetLogger() *slog.Logger {
	return log
}

// Enabled reports whether the handler handles records at the given level.
func (s *LogdocHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return s.Handler.Enabled(ctx, level)
}

// Handle intercepts and processes logger messages.
// In our case, send a message to the Logdoc.
func (s *LogdocHandler) Handle(ctx context.Context, record slog.Record) error {
	const (
		shortErrKey = "err"
		longErrKey  = "error"
	)

	if slices.Contains(s.levels, record.Level) {
		switch record.Level {
		case slog.LevelError:
			record.Attrs(func(attr slog.Attr) bool {
				if attr.Key == shortErrKey || attr.Key == longErrKey {
					if err, ok := attr.Value.Any().(error); ok {
						s.sendLogDocErrorEvent(err)
					}
				}

				return true
			})
		case slog.LevelDebug, slog.LevelInfo, slog.LevelWarn:
			s.sendLogDocEvent(record)
		}
	}

	return s.Handler.Handle(ctx, record)
}

// WithAttrs returns a new LogdocHandler whose attributes consists.
func (s *LogdocHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return NewLogdocHandler(s.Handler.WithAttrs(attrs), s.levels, s.proto, s.address, s.application)
}

// WithGroup returns a new LogdocHandler whose group consists.
func (s *LogdocHandler) WithGroup(name string) slog.Handler {
	return NewLogdocHandler(s.Handler.WithGroup(name), s.levels, s.proto, s.address, s.application)
}

func (s *LogdocHandler) sendLogDocEvent(entry slog.Record) {
	var lvl string
	if strings.Compare(entry.Level.String(), "warning") == 0 {
		lvl = "warn"
	} else {
		lvl = entry.Level.String()
	}

	go s.sendLogdoc(lvl, &entry, nil)
}

func (s *LogdocHandler) sendLogDocErrorEvent(err error) {
	go s.sendLogdoc(slog.LevelError.String(), nil, err)
}

func (s *LogdocHandler) sendLogdoc(level string, entry *slog.Record, err error) {
	header := []byte{6, 3}

	var msg string
	if entry != nil {
		msg = entry.Message
	} else {
		msg = err.Error()
	}

	app := s.application

	ip := s.Connection.RemoteAddr().String()
	pid := fmt.Sprintf("%d", os.Getpid())

	// TODO: обработать фреймы
	src := "TODO"
	// src := runtime.CallersFrames([]uintptr{entry.PC})

	t := time.Now()
	tsrc := t.Format("060201150405.000") + "\n"

	// Пишем заголовок
	result := header
	// Записываем само сообщение
	common.WritePair("msg", msg, &result)
	// Обрабатываем кастомные поля
	common.ProcessCustomFields(msg, &result)
	// Служебные поля
	common.WritePair("app", app, &result)
	common.WritePair("tsrc", tsrc, &result)
	common.WritePair("lvl", level, &result)
	common.WritePair("ip", ip, &result)
	common.WritePair("pid", pid, &result)
	common.WritePair("src", src, &result)

	// Финальный байт, завершаем
	result = append(result, []byte("\n")...)

	_, e := s.Connection.Write(result)
	if e != nil {
		log.Error("Ошибка записи в соединение, ", e)
	}

}

func networkWriter(proto string, address string) (net.Conn, error) {
	switch {
	case proto == "tcp":
		return tcpWriter(address)
	case proto == "udp":
		return udpWriter(address)
	default:
		log.With("address", address).Error("Error connecting LogDoc server")
		return nil, fmt.Errorf("error accessing LogDoc server, %s", address)
	}
}

// функция для создания TCP соединения и возврата io.Writer
func tcpWriter(address string) (net.Conn, error) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		log.With("address", address).Error("Error connecting LogDoc server using tcp")
		return nil, err
	}
	return conn, nil
}

// функция для создания UDP соединения и возврата io.Writer
func udpWriter(address string) (net.Conn, error) {
	conn, err := net.Dial("udp", address)
	if err != nil {
		log.With("address", address).Error("Error connecting LogDoc server using udp")
		return nil, err
	}
	return conn, nil
}
