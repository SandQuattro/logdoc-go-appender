package slogld

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// LogRecovery gin panic recovery with logging.
func LogRecovery(ctx *gin.Context, err interface{}) {
	reset := "\033[0m"

	brokenPipe := checkForBrokenConnection(err)

	if log != nil {
		stack := GetStack(3)
		httpRequest, _ := httputil.DumpRequest(ctx.Request, false)
		headers := strings.Split(string(httpRequest), "\r\n")
		for idx, header := range headers {
			current := strings.Split(header, ":")
			if current[0] == "Authorization" {
				headers[idx] = current[0] + ": *"
			}
		}
		headersToStr := strings.Join(headers, "\r\n")
		switch {
		case brokenPipe:
			log.Error(fmt.Sprintf("%s\n%s%s", err, headersToStr, reset))
		case gin.IsDebugging():
			if nilError, ok := err.(runtime.Error); ok {
				e := nilError.Error()
				log.
					With("headers", sanitize(headersToStr)).
					With("error", e).
					With("reset", sanitize(reset)).
					Error(fmt.Sprintf("[Recovery] %s panic recovered\nstack trace: %s", timeFormat(time.Now()), string(stack)))
			} else {
				log.
					With("headers", sanitize(headersToStr)).
					With("error", nilError).
					With("reset", sanitize(reset)).
					Error(fmt.Sprintf("[Recovery] %s panic recovered\nstack trace: %s", timeFormat(time.Now()), string(stack)))
			}
		default:
			log.Error(fmt.Sprintf("[Recovery] %s panic recovered:\n%s\n%s%s",
				timeFormat(time.Now()), err, stack, reset))
		}
	}

	if brokenPipe {
		// If the connection is dead, we can't write a status to it.
		if e, ok := err.(error); ok {
			ctx.Error(e) //nolint: errcheck
		}
		ctx.Abort()
	}

	// Respond with an error message
	ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Internal Server Error"})
}

// stack returns a nicely formatted stack frame, skipping skip frames.
func GetStack(skip int) []byte {
	buf := new(bytes.Buffer) // the returned data
	// As we loop, we open files and read them. These variables record the currently
	// loaded file.
	var lines [][]byte
	var lastFile string
	for i := skip; ; i++ { // Skip the expected number of frames
		programCounter, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		// Print this much at least.  If we can't find the source, it won't show.
		fmt.Fprintf(buf, "%s:%d (0x%x)\n", file, line, programCounter)
		if file != lastFile {
			data, err := os.ReadFile(filepath.Clean(file))
			if err != nil {
				continue
			}
			lines = bytes.Split(data, []byte{'\n'})
			lastFile = file
		}
		fmt.Fprintf(buf, "\t%s: %s\n", function(programCounter), source(lines, line))
	}

	return buf.Bytes()
}

// source returns a space-trimmed slice of the n'th line.
func source(lines [][]byte, nline int) []byte {
	dunno := []byte("???")

	nline-- // in stack trace, lines are 1-indexed but our array is 0-indexed
	if nline < 0 || nline >= len(lines) {
		return dunno
	}

	return bytes.TrimSpace(lines[nline])
}

// function returns, if possible, the name of the function containing the PC.
func function(pcall uintptr) []byte {
	dunno := []byte("???")
	centerDot := []byte("·")
	dot := []byte(".")
	slash := []byte("/")

	fn := runtime.FuncForPC(pcall)
	if fn == nil {
		return dunno
	}
	name := []byte(fn.Name())
	// The name includes the path name to the package, which is unnecessary
	// since the file name is already included.  Plus, it has center dots.
	// That is, we see
	//	runtime/debug.*T·ptrmethod
	// and want
	//	*T.ptrmethod
	// Also the package path might contain dot (e.g. code.google.com/...),
	// so first eliminate the path prefix
	if lastSlash := bytes.LastIndex(name, slash); lastSlash >= 0 {
		name = name[lastSlash+1:]
	}
	if period := bytes.Index(name, dot); period >= 0 {
		name = name[period+1:]
	}
	name = bytes.ReplaceAll(name, centerDot, dot)

	return name
}

// timeFormat returns a customized time string for log.
func timeFormat(t time.Time) string {
	return t.Format("2006/01/02 - 15:04:05")
}

func checkForBrokenConnection(err interface{}) bool {
	// Check for a broken connection, as it is not really a
	// condition that warrants a panic stack trace.
	var brokenPipe bool
	if ne, ok := err.(*net.OpError); ok {
		var se *os.SyscallError
		if errors.As(ne, &se) {
			seStr := strings.ToLower(se.Error())
			if strings.Contains(seStr, "broken pipe") ||
				strings.Contains(seStr, "connection reset by peer") {
				brokenPipe = true
			}
		}
	}

	return brokenPipe
}

func sanitize(str string) string {
	return strings.ReplaceAll(str, "\n", ", ")
}
