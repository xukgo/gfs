package model

type HttpError struct {
	error
	statusCode int
}

func InitHttpError(err error, code int) HttpError {
	return HttpError{
		error:      err,
		statusCode: code,
	}
}

func (err HttpError) StatusCode() int {
	return err.statusCode
}
func (err HttpError) Body() []byte {
	return []byte(err.Error())
}
