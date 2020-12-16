package shortener

type Error struct {
	ErrText string
	Origin  error
	Type    int
}

const (
	NotFoundErrType = iota
	InternalErrType
	BadParamsErrType
)

func (e Error) Error() string {
	return e.ErrText
}

func NewNotFoundError(errText string) error {
	return Error{
		ErrText: errText,
		Type:    NotFoundErrType,
	}
}

func NewInternalError(errText string, originErr error) error {
	return Error{
		ErrText: errText,
		Origin:  originErr,
		Type:    InternalErrType,
	}
}

func NewBadParamsError(errText string, originErr error) error {
	return Error{
		ErrText: errText,
		Origin:  originErr,
		Type:    BadParamsErrType,
	}
}
