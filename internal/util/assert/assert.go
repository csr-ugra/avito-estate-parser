package assert

import (
	"fmt"
	"github.com/csr-ugra/avito-estate-parser/internal/log"
	"os"
)

func assert(msg string, data ...interface{}) {
	fields := make(map[string]interface{})
	for i, d := range data {
		if i%2 == 0 {
			if i-1 < 0 {
				panic("assert args index out of range")
			}
			fields[fmt.Sprint(data[i-1])] = d
			continue
		}

		fields[fmt.Sprint(d)] = ""
	}

	logger := log.GetLogger()
	logger.WithFields(fields)

	logger.Fatal(msg)
	os.Exit(1)
}

func Assert(truth bool, msg string, data ...any) {
	if !truth {
		assert(msg, data...)
	}
}

func Nil(obj any, msg string, data ...any) {
	if obj == nil {
		return
	}

	assert(msg, data...)
}

func NotNil(obj any, msg string, data ...any) {
	if obj != nil {
		return
	}

	assert(msg, data...)
}

func Never(msg string, data ...any) {
	assert(msg, data...)
}

func NoError(err error, msg string, data ...any) {
	if err != nil {
		data = append(data, "error", err)
		assert(msg, data...)
	}
}
