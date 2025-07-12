package main

import (
	"log/slog"
	"net"
	"time"

	slogld "github.com/SandQuattro/logdoc-go-appender/slog"
)

func main() {
	// Создаем соединение с LogDoc
	conn, err := net.Dial("tcp", "localhost:9999")
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	// Настраиваем опции для slog handler
	option := slogld.Option{
		Level:       slog.LevelInfo,
		Conn:        conn,
		Application: "slog-example",
		AddSource:   true,
	}

	// Создаем handler и логгер
	handler := option.NewLogdocHandler()
	logger := slog.New(handler)

	// Устанавливаем как глобальный логгер
	slogld.SetLogger(logger)

	// Примеры использования
	logger.Info("Приложение запущено")

	logger.Info("Сервер запущен",
		slog.String("service", "api"),
		slog.Int("port", 8080),
	)

	logger.Warn("Попытка доступа к защищенному ресурсу",
		slog.String("user", "john"),
	)

	logger.Error("Ошибка выполнения запроса",
		slog.String("operation", "database_query"),
		slog.String("table", "users"),
		slog.String("error", "connection timeout"),
	)

	// Пример с дополнительными полями
	logger.Info("HTTP запрос обработан",
		slog.String("method", "GET"),
		slog.String("path", "/api/users"),
		slog.Int("status", 200),
		slog.Duration("duration", 150*time.Millisecond),
	)

	// Пример с группировкой полей
	logger.WithGroup("request").Info("Обработка запроса",
		slog.String("id", "req-123"),
		slog.String("client_ip", "192.168.1.100"),
	)

	// Пример с контекстными атрибутами
	contextLogger := logger.With(
		slog.String("session_id", "sess-456"),
		slog.String("user_id", "user-789"),
	)

	contextLogger.Info("Пользователь выполнил действие",
		slog.String("action", "create_document"),
		slog.String("document_id", "doc-101"),
	)

	// Пример с кастомными полями в сообщении
	logger.Info("Пользователь авторизован@@user=admin@role=superuser@ip=192.168.1.1")

	// Даем время для отправки логов
	time.Sleep(100 * time.Millisecond)
}
