package main

import (
	"time"

	logrusld "github.com/SandQuattro/logdoc-go-appender/logrus"
	"github.com/sirupsen/logrus"
)

func main() {
	// Инициализируем logrus с подключением к LogDoc
	conn, err := logrusld.Init("tcp", "localhost:9999", "logrus-example", logrusld.JSON)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	// Получаем логгер
	logger := logrusld.GetLogger()

	// Примеры использования
	logger.Info("Приложение запущено")

	logger.WithFields(logrus.Fields{
		"service": "api",
		"port":    8080,
	}).Info("Сервер запущен")

	logger.WithFields(logrus.Fields{
		"user": "john",
	}).Warn("Попытка доступа к защищенному ресурсу")

	logger.WithFields(logrus.Fields{
		"operation": "database_query",
		"table":     "users",
	}).Error("Ошибка выполнения запроса")

	// Пример с дополнительными полями
	logger.WithFields(logrus.Fields{
		"method":   "GET",
		"path":     "/api/users",
		"status":   200,
		"duration": "150ms",
	}).Info("HTTP запрос обработан")

	// Пример с кастомными полями в сообщении
	logger.Info("Пользователь авторизован@@user=admin@role=superuser@ip=192.168.1.1")

	// Даем время для отправки логов
	time.Sleep(100 * time.Millisecond)
}
