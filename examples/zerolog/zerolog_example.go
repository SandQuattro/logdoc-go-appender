package main

import (
	"time"

	zerologld "github.com/SandQuattro/logdoc-go-appender/zerolog"
	"github.com/rs/zerolog"
)

func main() {
	// Инициализируем zerolog с подключением к LogDoc
	conn, err := zerologld.Init("tcp", "localhost:9999", "zerolog-example", zerolog.InfoLevel, zerologld.JSON)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	// Получаем логгер
	logger := zerologld.GetLogger()

	// Примеры использования
	logger.Info().Msg("Приложение запущено")

	logger.Info().
		Str("service", "api").
		Int("port", 8080).
		Msg("Сервер запущен")

	logger.Warn().
		Str("user", "john").
		Msg("Попытка доступа к защищенному ресурсу")

	logger.Error().
		Err(err).
		Str("operation", "database_query").
		Msg("Ошибка выполнения запроса")

	// Пример с дополнительными полями
	logger.Info().
		Str("method", "GET").
		Str("path", "/api/users").
		Int("status", 200).
		Dur("duration", 150*time.Millisecond).
		Msg("HTTP запрос обработан")

	// Даем время для отправки логов
	time.Sleep(100 * time.Millisecond)
}
