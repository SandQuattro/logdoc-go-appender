package main

import (
	"time"

	zapld "github.com/SandQuattro/logdoc-go-appender/zap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	// Инициализируем zap с подключением к LogDoc
	conn, err := zapld.Init(nil, zapcore.InfoLevel, "tcp", "localhost:9999", "zap-example", false)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	// Получаем логгер
	logger := zapld.GetLogger()

	// Примеры использования
	logger.Info("Приложение запущено")

	logger.Info("Сервер запущен",
		zap.String("service", "api"),
		zap.Int("port", 8080),
	)

	logger.Warn("Попытка доступа к защищенному ресурсу",
		zap.String("user", "john"),
	)

	logger.Error("Ошибка выполнения запроса",
		zap.String("operation", "database_query"),
		zap.String("table", "users"),
		zap.Error(err),
	)

	// Пример с дополнительными полями
	logger.Info("HTTP запрос обработан",
		zap.String("method", "GET"),
		zap.String("path", "/api/users"),
		zap.Int("status", 200),
		zap.Duration("duration", 150*time.Millisecond),
	)

	// Пример с кастомными полями в сообщении
	logger.Info("Пользователь авторизован@@user=admin@role=superuser@ip=192.168.1.1")

	// Пример с SugaredLogger для более удобного API
	sugar := logger.Sugar()
	sugar.Infow("Событие с именованными параметрами",
		"event", "user_login",
		"user_id", 12345,
		"success", true,
	)

	sugar.Infof("Пользователь %s выполнил действие %s", "admin", "create_user")

	// Даем время для отправки логов
	time.Sleep(100 * time.Millisecond)
}
