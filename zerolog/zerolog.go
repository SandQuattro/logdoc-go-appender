package zerologld

import (
	"fmt"
	"io"
	"net"
	"os"
	"runtime"
	"strconv"

	"github.com/SandQuattro/logdoc-go-appender/common"
	"github.com/rs/zerolog"
)

// Константы для выбора формата вывода
const (
	JSON = 0
	TEXT = 1
)

var application string
var connection net.Conn
var log *zerolog.Logger

// init инициализирует логгер по умолчанию с уровнем debug
func init() {
	defaultLogger := zerolog.New(os.Stdout).
		With().
		Timestamp().
		Logger().
		Level(zerolog.DebugLevel)
	log = &defaultLogger
}

// LogdocHook реализует интерфейс zerolog.Hook для отправки логов в LogDoc
type LogdocHook struct {
	Conn        net.Conn
	Application string
}

// Run выполняется для каждого лог события
func (h LogdocHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
	// Проверяем наличие соединения
	if h.Conn == nil {
		return
	}

	// Получаем информацию о вызывающем коде
	pc, _, line, ok := runtime.Caller(3) // 3 уровня вверх по стеку
	var src string
	if ok {
		f := runtime.FuncForPC(pc)
		if f != nil {
			src = f.Name() + ":" + strconv.Itoa(line)
		} else {
			src = "unknown"
		}
	} else {
		src = "unknown"
	}

	// Используем общую функцию для обработки кастомных полей
	customFieldsProcessor := func(result *[]byte) {
		// В zerolog кастомные поля уже сериализованы в msg
		// Поэтому используем стандартную обработку
		common.ProcessCustomFields(msg, result)
	}

	result := common.BuildLogDocMessage(
		msg,
		h.Application,
		level.String(),
		src,
		h.Conn,
		customFieldsProcessor,
	)

	// Отправляем асинхронно
	go func() {
		_, err := h.Conn.Write(result)
		if err != nil {
			log.Error().Err(err).Msg("Ошибка записи в соединение LogDoc")
		}
	}()
}

// GetLogger возвращает текущий логгер
func GetLogger() *zerolog.Logger {
	return log
}

// SetLogger устанавливает новый логгер
func SetLogger(logger *zerolog.Logger) {
	log = logger
}

// Init инициализирует zerolog с подключением к LogDoc
func Init(proto string, address string, app string, level zerolog.Level, format int) (net.Conn, error) {
	application = app

	writer := io.Writer(os.Stdout)
	if format == TEXT {
		writer = zerolog.ConsoleWriter{Out: os.Stdout}
	}

	logger := zerolog.New(writer).
		With().
		Timestamp().
		Logger().
		Level(level)

	SetLogger(&logger)

	conn, err := networkWriter(proto, address)
	if err != nil {

	}
	hook := LogdocHook{
		Conn:        conn,
		Application: app,
	}

	if conn != nil {
		logger = logger.Hook(hook)
	}

	return conn, nil
}

// networkWriter создает сетевое соединение
func networkWriter(proto string, address string) (net.Conn, error) {
	switch proto {
	case "tcp":
		return tcpWriter(address)
	case "udp":
		return udpWriter(address)
	default:
		return nil, fmt.Errorf("неподдерживаемый протокол: %s", proto)
	}
}

// tcpWriter создает TCP соединение
func tcpWriter(address string) (net.Conn, error) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("ошибка TCP соединения с %s: %w", address, err)
	}
	return conn, nil
}

// udpWriter создает UDP соединение
func udpWriter(address string) (net.Conn, error) {
	conn, err := net.Dial("udp", address)
	if err != nil {
		return nil, fmt.Errorf("ошибка UDP соединения с %s: %w", address, err)
	}
	return conn, nil
}
