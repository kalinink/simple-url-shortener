package repository

import (
	"context"
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/kalinink/simple-url-shortener/internal/shortener"
	"time"
)

type URL struct {
	db      *sqlx.DB
	timeout time.Duration
}

func NewURL(db *sqlx.DB, dbTimeout time.Duration) *URL {
	return &URL{db: db, timeout: dbTimeout}
}

func (repo *URL) Save(ctx context.Context, url *shortener.NewURL) error {
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, repo.timeout)
	defer cancel()

	query := "INSERT INTO urls (short_url, origin, created_at) VALUES ($1, $2, $3)"

	_, err := repo.db.ExecContext(ctx, query, &url.Short, &url.Long, &url.CreatedAt)
	if err != nil {
		return toServiceError(err)
	}

	return nil
}

func (repo *URL) GetIfNotExpired(ctx context.Context, url *shortener.ShortURL, expiredURL shortener.CheckExpiredFunc) (*shortener.URL, error) {
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, repo.timeout)
	defer cancel()

	u, err := repo.getURL(ctx, url.URL)
	if err != nil {
		return nil, toServiceError(err)
	}

	if expiredURL(u.LastAccess, u.CreatedAt) {
		if err := repo.setURLExpired(ctx, u.ShortURL); err != nil {
			return nil, toServiceError(err)
		}

		return nil, shortener.NewNotFoundError("url not found")
	}

	if err := repo.updateAccess(ctx, u.ShortURL, url.AccessTime); err != nil {
		return nil, toServiceError(err)
	}

	return &shortener.URL{
		Long:  u.Origin,
		Short: u.ShortURL,
	}, nil

}

func (repo *URL) IncShort(ctx context.Context) error {
	if err := repo.addAccessRow(ctx, "short_urls_access", time.Now()); err != nil {
		return toServiceError(err)
	}

	return nil
}

func (repo *URL) IncLong(ctx context.Context) error {
	if err := repo.addAccessRow(ctx, "long_urls_access", time.Now()); err != nil {
		return toServiceError(err)
	}

	return nil
}

func (repo *URL) StatShortURL(ctx context.Context) (*shortener.Statistics, error) {
	s, err := repo.stat(ctx, "short_urls_access")
	if err != nil {
		return nil, toServiceError(err)
	}

	stats := shortener.Statistics{Timing: s.Timing}
	if s.Count != nil {
		stats.Count = *s.Count
	}

	return &stats, nil
}

func (repo *URL) StatLongURL(ctx context.Context) (*shortener.Statistics, error) {
	s, err := repo.stat(ctx, "long_urls_access")
	if err != nil {
		return nil, toServiceError(err)
	}

	stats := shortener.Statistics{Timing: s.Timing}
	if s.Count != nil {
		stats.Count = *s.Count
	}

	return &stats, nil
}

func (repo *URL) updateAccess(ctx context.Context, shortURL string, t time.Time) error {
	query := "UPDATE urls SET last_access = $1 WHERE short_url = $2"
	_, err := repo.db.ExecContext(ctx, query, &t, &shortURL)
	return err
}

func (repo *URL) setURLExpired(ctx context.Context, shortURL string) error {
	query := "UPDATE urls SET is_expired = true WHERE short_url = $1"
	_, err := repo.db.ExecContext(ctx, query, &shortURL)
	return err
}

func (repo *URL) getURL(ctx context.Context, shortURL string) (*URLs, error) {
	query := `
		SELECT short_url, origin, created_at, last_access
		FROM urls
		WHERE is_expired = false AND short_url = $1
	`

	u := URLs{}
	if err := repo.db.QueryRowxContext(ctx, query, &shortURL).StructScan(&u); err != nil {
		return nil, err
	}

	return &u, nil
}

func (repo *URL) addAccessRow(ctx context.Context, table string, t time.Time) error {
	query := fmt.Sprintf("INSERT INTO %s (access_at) VALUES($1)", table)
	_, err := repo.db.ExecContext(ctx, query, &t)
	return err
}

func (repo *URL) stat(ctx context.Context, table string) (*Statistics, error) {
	query := fmt.Sprintf(`
		SELECT access_at, count
		FROM (
		    SELECT
		           access_at,
		           (select count(*) from %s) as count,
		           ROW_NUMBER() OVER (ORDER BY access_at) as rank
		            FROM %s
		    ) as ranks
		WHERE rank = count/2;
	`, table, table)

	s := Statistics{}
	if err := repo.db.QueryRowxContext(ctx, query).StructScan(&s); err != nil {
		return nil, err
	}

	return &s, nil
}
