package repository

import "time"

type URLs struct {
	ShortURL   string     `db:"short_url"`
	Origin     string     `db:"origin"`
	CreatedAt  time.Time  `db:"created_at"`
	LastAccess *time.Time `db:"last_access"`
	IsExpired  bool       `db:"is_expired"`
}

type URLsAccess struct {
	AccessAt time.Time `db:"access_at"`
}

type Statistics struct {
	Timing *time.Time `db:"access_at"`
	Count  *int       `db:"count"`
}
