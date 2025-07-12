package common

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

func WritePair(key string, value string, arr *[]byte) {
	sepIdx := strings.Index(value, "@@")
	msg := ""
	if sepIdx != -1 {
		msg = value[0:sepIdx]
	} else {
		msg = value
	}
	if strings.Contains(msg, "\n") {
		writeComplexPair(key, msg, arr)
	} else {
		writeSimplePair(key, msg, arr)
	}
}

func writeComplexPair(key string, value string, arr *[]byte) {
	*arr = append(*arr, []byte(key)...)
	*arr = append(*arr, []byte("\n")...)
	*arr = append(*arr, writeInt(len(value))...)
	*arr = append(*arr, []byte(value)...)
}

func writeSimplePair(key string, value string, arr *[]byte) {
	*arr = append(*arr, []byte(key+"="+value+"\n")...)
}

func ProcessCustomFields(msg string, arr *[]byte) {
	// Обработка кастом полей
	sepIdx := strings.Index(msg, "@@")
	rawFields := ""

	if sepIdx != -1 {
		rawFields = msg[sepIdx+2:]
		keyValuePairs := strings.Split(rawFields, "@")

		for _, pair := range keyValuePairs {
			keyValue := strings.Split(pair, "=")
			if len(keyValue) == 2 {
				*arr = append(*arr, []byte(keyValue[0]+"="+keyValue[1]+"\n")...)
			}
		}
	}
}

func writeInt(in int) []byte {
	buf := new(bytes.Buffer)
	buf.WriteByte(byte((in >> 24) & 0xff))
	buf.WriteByte(byte((in >> 16) & 0xff))
	buf.WriteByte(byte((in >> 8) & 0xff))
	buf.WriteByte(byte(in & 0xff))
	return buf.Bytes()
}

func SourceNameWithLine(pc uintptr, file string, line int, ok bool) string {
	return fmt.Sprintf("%s:%d", GetSourceName(pc, file, line, ok), line)
}

func GetSourceName(pc uintptr, file string, line int, ok bool) string {
	// in skip if we're using 1, so it will actually log the where the error happened, 0 = this function
	return file[strings.LastIndex(file, "/")+1:]
}

func GetSourceLineNum(pc uintptr, file string, line int, ok bool) int {
	// in skip if we're using 1, so it will actually log the where the error happened, 0 = this function
	return line
}

// NormalizeLogLevel нормализует уровень логирования (warning -> warn)
func NormalizeLogLevel(level string) string {
	if strings.Compare(level, "warning") == 0 {
		return "warn"
	}
	return level
}

// FormatTimestamp форматирует временную метку для LogDoc
func FormatTimestamp() string {
	t := time.Now()
	tsrc := t.Format("060102150405.000") + "\n"
	return strings.ReplaceAll(tsrc, ".", "")
}

// GetLogDocHeader возвращает стандартный заголовок для LogDoc
func GetLogDocHeader() []byte {
	return []byte{6, 3}
}

// GetProcessInfo возвращает информацию о процессе (PID и IP)
func GetProcessInfo(conn net.Conn) (string, string) {
	pid := fmt.Sprintf("%d", os.Getpid())
	ip := conn.RemoteAddr().String()
	return pid, ip
}

// BuildLogDocMessage строит полное сообщение для LogDoc
func BuildLogDocMessage(msg, app, level, src string, conn net.Conn, customFieldsProcessor func(*[]byte)) []byte {
	header := GetLogDocHeader()
	pid, ip := GetProcessInfo(conn)
	tsrc := FormatTimestamp()
	normalizedLevel := NormalizeLogLevel(level)

	// Пишем заголовок
	result := header
	// Записываем само сообщение
	WritePair("msg", msg, &result)
	// Обрабатываем кастомные поля
	if customFieldsProcessor != nil {
		customFieldsProcessor(&result)
	}
	// Служебные поля
	WritePair("app", app, &result)
	WritePair("tsrc", tsrc, &result)
	WritePair("lvl", normalizedLevel, &result)
	WritePair("ip", ip, &result)
	WritePair("pid", pid, &result)
	WritePair("src", src, &result)

	// Финальный байт, завершаем
	result = append(result, []byte("\n")...)

	return result
}
