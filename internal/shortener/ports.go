package shortener

import (
	"context"
)

type URLShortenerService interface {
	GetLongURL(ctx context.Context, shortURL string) (*URL, error)
	CreateShortURL(ctx context.Context, longURL string) (*URL, error)
	Statistics(context.Context) (*OverallStatistics, error)
}

type URLRepository interface {
	Save(context.Context, *NewURL) error
	GetIfNotExpired(context.Context, *ShortURL, checkExpiredFunc) (*URL, error)
	IncShort(context.Context) error
	IncLong(context.Context) error
	StatShortURL(context.Context) (*Statistics, error)
	StatLongURL(context.Context) (*Statistics, error)
}
