package error

type Error struct {
	Message string
	Args    []interface{}
}

func New(message string, args ...interface{}) Error {
	return Error{Message: message, Args: args}
}
