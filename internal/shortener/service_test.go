package shortener

import (
	"context"
	"fmt"
	"github.com/rs/zerolog"
	"sync"
	"testing"
	"time"
)

const (
	hostName = "example.com"
	scheme   = "http"
	noError  = -1
)

func TestService_GetLongURL(t *testing.T) {
	srv := newTestService(500 * time.Millisecond)
	cases := []struct {
		longURL  string
		shortURL string
		errType  int
	}{
		{
			longURL: "https://stackoverflow.com/questions/65324815/issorted-check-array-if-is-sorted-in-ascending-or-descending-order-in-java",
			errType: noError,
		},
		{
			longURL:  "http://wrong:url:s",
			errType:  BadParamsErrType,
			shortURL: fmt.Sprintf("%s://%s", scheme, hostName),
		},
		{
			longURL: "http://wiki.amperka.ru/products:pimoroni-raspberry-pi-enviro-air-quality",
			errType: noError,
		},
	}
	ctx := context.Background()
	for i := range cases {
		u, err := srv.CreateShortURL(ctx, cases[i].longURL)
		if cases[i].errType != noError {
			AssertError(t, err, cases[i].errType, fmt.Sprintf("case #%d", i))
		} else {
			AssertNoError(t, err, fmt.Sprintf("case #%d", i))
			cases[i].shortURL = u.Short
		}
	}

	for i := range cases {
		_, err := srv.GetLongURL(ctx, cases[i].shortURL)
		if cases[i].errType != noError {
			AssertError(t, err, NotFoundErrType, fmt.Sprintf("case #%d", i))
		} else {
			AssertNoError(t, err, fmt.Sprintf("case #%d", i))
		}
	}
}

func TestService_GetExpired(t *testing.T) {
	expiredAfter := 100 * time.Millisecond
	srv := newTestService(expiredAfter)
	data := struct {
		longURL  string
		shortURL string
		errType  int
	}{
		longURL: "https://stackoverflow.com/questions/65324815/issorted-check-array-if-is-sorted-in-ascending-or-descending-order-in-java",
		errType: noError,
	}

	ctx := context.Background()
	u, err := srv.CreateShortURL(ctx, data.longURL)
	AssertNoError(t, err, "creation short url")
	data.shortURL = u.Short

	_, err = srv.GetLongURL(ctx, data.shortURL)
	AssertNoError(t, err, "getting not expired url")

	time.Sleep(expiredAfter)
	_, err = srv.GetLongURL(ctx, data.shortURL)
	AssertError(t, err, NotFoundErrType, "getting expired url")
}

func TestService_Statistics(t *testing.T) {
	srv := newTestService(500 * time.Millisecond)
	data := struct {
		longURL  string
		shortURL string
		errType  int
	}{
		longURL: "https://stackoverflow.com/questions/65324815/issorted",
		errType: noError,
	}

	ctx := context.Background()
	u, err := srv.CreateShortURL(ctx, data.longURL)
	AssertNoError(t, err, "creation short url")
	data.shortURL = u.Short

	reqNumber := 10
	for i := 0; i < reqNumber; i++ {
		_, err = srv.GetLongURL(ctx, data.shortURL)
		AssertNoError(t, err, "getting url")
		time.Sleep(100 * time.Millisecond)
	}

	stat, err := srv.Statistics(ctx)
	AssertNoError(t, err, "getting statistic")

	if stat.ShortURL.Count != 1 {
		t.Errorf("wa")
	}
	if stat.LongURL.Count != reqNumber {
		t.Errorf("want %d, got %d", reqNumber, stat.LongURL.Count)
	}
}

type inMemoryDB struct {
	mu             sync.Mutex
	store          map[string]row
	shortStatStore []time.Time
	longStatStore  []time.Time
}

func newInMemoryDB() *inMemoryDB {
	return &inMemoryDB{store: make(map[string]row)}
}

type row struct {
	longURL    string
	lastAccess *time.Time
	createdAt  time.Time
}

func (db *inMemoryDB) Save(ctx context.Context, url *NewURL) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.store[url.Short] = row{longURL: url.Long, createdAt: url.CreatedAt}
	return nil
}

func (db *inMemoryDB) GetIfNotExpired(ctx context.Context, s *ShortURL, isExpired checkExpiredFunc) (*URL, error) {
	u, exists := db.store[s.URL]
	if !exists {
		return nil, NewNotFoundError("url not found")
	}

	if isExpired(u.lastAccess, u.createdAt) {
		return nil, NewNotFoundError("url not found")
	}

	db.mu.Lock()
	defer db.mu.Unlock()

	u.lastAccess = &s.AccessTime
	db.store[s.URL] = u

	return &URL{Long: u.longURL, Short: s.URL}, nil
}

func (db *inMemoryDB) IncShort(ctx context.Context) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.shortStatStore = append(db.shortStatStore, time.Now())
	return nil
}

func (db *inMemoryDB) IncLong(ctx context.Context) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.longStatStore = append(db.longStatStore, time.Now())
	return nil
}

func (db *inMemoryDB) StatShortURL(ctx context.Context) (*Statistics, error) {
	return stat(db.shortStatStore), nil
}

func (db *inMemoryDB) StatLongURL(ctx context.Context) (*Statistics, error) {
	return stat(db.longStatStore), nil
}

func stat(arr []time.Time) *Statistics {
	count := len(arr)
	median := arr[count/2]
	return &Statistics{
		Count:  count,
		Timing: median,
	}
}

func newTestService(expired time.Duration) *Service {
	repo := newInMemoryDB()
	log := zerolog.New(nil).With().Logger()
	return NewService(repo, hostName, scheme, expired, &log)
}

func AssertNoError(t *testing.T, got error, name string) {
	t.Helper()
	if got != nil {
		t.Fatalf("[%s] got an error but didn't want one: %+v", name, got)
	}
}

func AssertError(t *testing.T, got error, wantErrType int, name string) {
	t.Helper()
	if got == nil {
		t.Fatalf("[%s] didn't get an error but wanted", name)
	}

	urlErr, ok := got.(Error)
	if !ok {
		t.Errorf("[%s] want err type %T, got %T", name, Error{}, got)
		return
	}

	if urlErr.Type != wantErrType {
		t.Errorf("[%s] got %d errType (%s), want %d errType", name, urlErr.Type, urlErr, wantErrType)
	}
}
