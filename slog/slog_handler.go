package slogld

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/SandQuattro/logdoc-go-appender/common"
	slogcommon "github.com/samber/slog-common"
)

var _ slog.Handler = (*LogdocHandler)(nil)

var log = slog.Default()

type Option struct {
	// log level (default: debug)
	Level slog.Leveler

	// connection to logdoc
	Conn net.Conn

	// application name in logdoc
	app string

	// optional: customize json payload builder
	Converter Converter

	// optional: custom marshaler
	Marshaler func(v any) ([]byte, error)

	// optional: fetch attributes from context
	AttrFromContext []func(ctx context.Context) []slog.Attr

	// optional: see slog.HandlerOptions
	AddSource   bool
	ReplaceAttr func(groups []string, a slog.Attr) slog.Attr
}

// LogdocHandler is a Handler that writes log records to the Logdoc.
type LogdocHandler struct {
	option Option
	attrs  []slog.Attr
	groups []string
}

// NewLogdocHandler creates a LogdocHandler using the given option.
func (o Option) NewLogdocHandler() slog.Handler {
	if o.Level == nil {
		o.Level = slog.LevelDebug
	}

	if o.Conn == nil {
		panic("missing logdoc connection")
	}

	if o.Converter == nil {
		o.Converter = DefaultConverter
	}

	if o.Marshaler == nil {
		o.Marshaler = json.Marshal
	}

	if o.AttrFromContext == nil {
		o.AttrFromContext = []func(ctx context.Context) []slog.Attr{}
	}

	return &LogdocHandler{
		option: o,
		attrs:  []slog.Attr{},
		groups: []string{},
	}
}

func GetLogger() *slog.Logger {
	return log
}

func SetLogger(logger *slog.Logger) {
	log = logger
}

func (h *LogdocHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.option.Level.Level()
}

// Handle intercepts and processes logger messages.
// In our case, send a message to the Logdoc.
func (h *LogdocHandler) Handle(ctx context.Context, record slog.Record) error {
	fromContext := slogcommon.ContextExtractor(ctx, h.option.AttrFromContext)
	message := h.option.Converter(h.option.AddSource, h.option.ReplaceAttr, append(h.attrs, fromContext...), h.groups, &record)

	go func() {
		h.sendLogDocEvent(message)
	}()

	return nil
}

func (h *LogdocHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &LogdocHandler{
		option: h.option,
		attrs:  slogcommon.AppendAttrsToGroup(h.groups, h.attrs, attrs...),
		groups: h.groups,
	}
}

func (h *LogdocHandler) WithGroup(name string) slog.Handler {
	return &LogdocHandler{
		option: h.option,
		attrs:  h.attrs,
		groups: append(h.groups, name),
	}
}

func (h *LogdocHandler) sendLogDocEvent(entry *slog.Record) {
	var lvl string
	if strings.Compare(entry.Level.String(), "warning") == 0 {
		lvl = "warn"
	} else {
		lvl = entry.Level.String()
	}

	go h.sendLogdoc(lvl, entry, nil)
}

func (h *LogdocHandler) sendLogdoc(level string, entry *slog.Record, err error) {
	header := []byte{6, 3}

	var msg string
	if entry != nil {
		msg = entry.Message
	} else {
		msg = err.Error()
	}

	app := h.option.app

	ip := h.option.Conn.RemoteAddr().String()
	pid := fmt.Sprintf("%d", os.Getpid())

	var src string
	if entry != nil {
		f := runtime.FuncForPC(entry.PC)
		_, line := f.FileLine(entry.PC)
		src = f.Name() + ":" + strconv.Itoa(line)
	} else {
		// TODO: обработать фреймы ошибки
		src = "TODO"
	}

	t := time.Now()
	tsrc := t.Format("060201150405.000") + "\n"

	// Пишем заголовок
	result := header
	// Записываем само сообщение
	common.WritePair("msg", msg, &result)
	// Обрабатываем кастомные поля
	result = processCustomFields(entry, result)
	// Служебные поля
	common.WritePair("app", app, &result)
	common.WritePair("tsrc", tsrc, &result)
	common.WritePair("lvl", level, &result)
	common.WritePair("ip", ip, &result)
	common.WritePair("pid", pid, &result)
	common.WritePair("src", src, &result)

	// Финальный байт, завершаем
	result = append(result, []byte("\n")...)

	_, e := h.option.Conn.Write(result)
	if e != nil {
		log.Error("Ошибка записи в соединение, ", e)
	}

}

func processCustomFields(record *slog.Record, result []byte) []byte {
	// Обработка кастом полей
	record.Attrs(func(attr slog.Attr) bool {
		key, val := slogcommon.AttrToValue(attr)
		if v, ok := val.(string); ok {
			result = append(result, []byte(key+"="+v+"\n")...)
		}
		return true
	})

	return result
}
