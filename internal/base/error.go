package base

type ServerError struct {
	msg string
}

func (e *ServerError) Error() string {
	return e.msg
}

func NewServerError(msg string) error {
	return &ServerError{msg: msg}
}

var (
	ErrDatabaseQuery = NewServerError("db query failed")
)

type LogicError struct {
	msg string
}

func (e *LogicError) Error() string {
	return e.msg
}

func NewLogicError(msg string) error {
	return &LogicError{msg: msg}
}

var (
	ErrTaskNotFound = NewLogicError("task not found")
	ErrTagNotFound  = NewLogicError("tag not found")
)
