package textio

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
)

// type ReaderErrorKind int

var (
	ErrInvalid = errors.New("textio: invalid token")
	ErrRead    = errors.New("textio: read error")
	ErrClose   = errors.New("textio: close error")
)

type ReaderError struct {
	Kind error
	Err  error
	// Metadata
	Token     string
	Index     int
	FileName  string
	FuncName  string
	ErrorLine int
}

func (e *ReaderError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%v: %v", e.Kind, e.Err)
	}
	return e.Kind.Error()
}

func (e *ReaderError) Is(target error) bool {
	return e.Kind == target
}

func (e *ReaderError) Unwrap() error {
	return e.Err
}

func newReaderError(skip int) *ReaderError {
	pc, file, line, _ := runtime.Caller(skip)

	fileName := file
	if i := strings.LastIndex(file, "/"); i >= 0 {
		fileName = file[i+1:]
	}

	funcName := ""
	if fn := runtime.FuncForPC(pc); fn != nil {
		name := fn.Name()
		if i := strings.LastIndex(name, "."); i >= 0 {
			funcName = name[i+1:]
		} else {
			funcName = name
		}
	}

	return &ReaderError{
		FileName:  fileName,
		FuncName:  funcName,
		ErrorLine: line,
		Index:     -1,
	}
}

func newErrInvalid(token string, index int) error {
	re := newReaderError(3)
	re.Kind = ErrInvalid
	re.Token = token
	re.Index = index
	return re
}

func newErrRead(err error) error {
	re := newReaderError(3)
	re.Kind = ErrRead
	re.Err = err
	return re
}
