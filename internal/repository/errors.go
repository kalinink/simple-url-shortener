package repository

import (
	"database/sql"
	"github.com/kalinink/simple-url-shortener/internal/shortener"

	"github.com/jackc/pgerrcode"
	"github.com/lib/pq"
)

type PGError struct {
	Origin error
	Type   string
}

func (e PGError) Error() string {
	return e.Origin.Error()
}

func NewPGError(err error, t string) PGError {
	return PGError{Origin: err, Type: t}
}

const (
	Unknown             = "unknown error"
	NotFound            = "not found"
	UniqueViolation     = "unique violation"
	ForeignKeyViolation = "foreign key violation"
)

func handlePGError(err error) error {
	if err == sql.ErrNoRows {
		return NewPGError(err, NotFound)
	}
	if pgErr, ok := err.(*pq.Error); ok {
		switch pgErr.Code {
		case pgerrcode.UniqueViolation:
			return NewPGError(err, UniqueViolation)
		case pgerrcode.ForeignKeyViolation:
			return NewPGError(err, ForeignKeyViolation)
		}
	}
	return NewPGError(err, Unknown)
}

func toServiceError(err error) error {
	pgErr := handlePGError(err).(PGError)

	errType := shortener.InternalErrType
	var text string

	switch pgErr.Type {
	case NotFound:
		errType = shortener.NotFoundErrType
		text = "not found"
	case UniqueViolation, ForeignKeyViolation:
		errType = shortener.BadParamsErrType
	}

	return shortener.Error{Type: errType, Origin: pgErr, ErrText: text}
}
