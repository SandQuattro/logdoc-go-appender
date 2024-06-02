[![Go Report Card](https://goreportcard.com/badge/github.com/SandQuattro/logdoc-go-appender)](https://goreportcard.com/report/github.com/SandQuattro/logdoc-go-appender)

# logdoc-go-appender

## Logdoc Go Appender v0.0.27

### Структурированные логи в GoLang
[sirupsen/logrus](https://github.com/uber-go/zap) — один из самых популярных пакетов логирования, использующий структурированные логи JSON.

[uber-go/zap](https://github.com/uber-go/zap) — супербыстрый логгер со структурированными логами JSON.

rs/zerolog — очень быстрые структурированные логи в формате JSON.

slog - высокопроизводительный пакет для структурированного ведения логов в стандартной библиотеке Go

### Рассмотрим подробнее sirupsen/logrus, uber-go/zap и встроенный в golang 1.21 slog
Это популярные библиотеки логирования для языка программирования Go.

logrus - это библиотека с открытым исходным кодом, предназначенная для логирования в Go. Она предоставляет множество функций для логирования, таких как логирование на уровне отладки, информации, предупреждения и ошибок. Она также поддерживает различные форматы логирования, такие как JSON, текст и т.д. Logrus также имеет множество плагинов, которые позволяют настраивать логирование под конкретные нужды.

Zap - это библиотека с открытым исходным кодом, которая была разработана Uber для логирования в Go. Она была создана для обеспечения высокой производительности и эффективности. Она предоставляет множество функций для логирования, таких как логирование на уровне отладки, информации, предупреждения и ошибок. Она также поддерживает различные форматы логирования, такие как JSON, текст и т.д. Zap также имеет множество плагинов, которые позволяют настраивать логирование под конкретные нужды.

slog - В 2023 году, команда разработчиков Google Go наконец-то представила Slog — высокопроизводительный пакет для структурированного ведения логов в стандартной библиотеке Go

### Использование логгеров
плагин logdoc-go-appender в данный момент использует logrus,zap и slog, для передачи логов на LogDoc server, используя LogDoc Native Protocol

### Как подключить в свой проект, пример с logrus
В раздел import добавляем пакет logrusld "github.com/SandQuattro/logdoc-go-appender/logrus", запускаем sync библиотек из среды разработки, в терминале go get -u github.com/SandQuattro/logdoc-go-appender или вводим в терминале go mod tidy (tidy удостоверяется, что go.mod соответствует исходному коду в модуле. Он добавляет все недостающие модули, необходимые для построения пакетов и зависимостей текущего модуля, и удаляет неиспользуемые модули, которые не предоставляют никаких соответствующих пакетов. Он также добавляет все недостающие записи в go.sum и удаляет ненужные)

Далее, в main.go, в начале приложения инициализируем подсистему логирования LogDoc:

```go
import logrusld "github.com/SandQuattro/logdoc-go-appender/logrus"

...

// Создаем подсистему логгирования LogDoc
	conn, err := LDSubsystemInit()
	logger := logrusld.GetLogger()
	if err == nil {
		logger.Info(fmt.Sprintf(
			"LogDoc subsystem initialized successfully@@source=%s:%d",
			logdoc.GetSourceName(runtime.Caller(0)), // фреймы не скипаем, не exception
			logdoc.GetSourceLineNum(runtime.Caller(0)),
		))
	}

	c := *conn
	if c != nil {
		defer c.Close()
	} else {
		logger.Error("Error LogDoc subsystem initialization")
	}
...
func LDSubsystemInit() (*net.Conn, error) {
	conf := config.GetConfig()
	conn, err := logrusld.Init(
		conf.GetString("ld.proto"),
		conf.GetString("ld.host")+":"+conf.GetString("ld.port"),
		conf.GetString("ld.app"), 
		logdoc.TEXT, // logdoc.JSON,
	)
	return &conn, err
}

```

Здесь я использую конфигурацию приложения с использованием HOCON и библиотеки 
"github.com/gurkankaymak/hocon", но здесь вы можете использовать любую конфигурацию, главное - инициализировать LogDoc:

```go
conn, err := logrusld.Init("tcp или udp","host:port", "название вашего приложения", logdoc.TEXT)
```

Далее в любом модуле необходимо получить логгер: logger := logrusld.GetLogger() и пользоваться им, как обычным logrus:

```go
logger.Error("Тут возникла ошибка", err)
logger.Debug("Отладочное сообщение")
```
