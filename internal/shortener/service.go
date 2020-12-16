package shortener

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"github.com/rs/zerolog"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const shortURLPathLength = 12

type checkExpiredFunc func(lastAccess *time.Time, createdAt time.Time) bool

type Service struct {
	urlRepository URLRepository
	scheme        string
	hostName      string
	expiredAfter  time.Duration
	log           *zerolog.Logger
}

func NewService(repo URLRepository, hostName string, scheme string, expired time.Duration, log *zerolog.Logger) *Service {
	return &Service{
		urlRepository: repo,
		scheme:        scheme,
		hostName:      hostName,
		expiredAfter:  expired,
		log:           log,
	}
}

func (srv *Service) CreateShortURL(ctx context.Context, longURL string) (*URL, error) {
	_, err := url.Parse(longURL)
	if err != nil {
		return nil, NewBadParamsError("invalid url format", err)
	}

	shortURL := srv.makeShortURL(longURL)

	err = srv.urlRepository.Save(ctx, &NewURL{
		Long:      longURL,
		Short:     shortURLKey(shortURL),
		CreatedAt: time.Now(),
	})

	if err != nil {
		return nil, err
	}

	if err := srv.urlRepository.IncShort(ctx); err != nil {
		srv.log.Err(err).Msg("the attempt to increase the count of 'short' calls")
	}

	return &URL{
		Long:  longURL,
		Short: shortURL.String(),
	}, nil
}

func (srv *Service) GetLongURL(ctx context.Context, shortURL string) (*URL, error) {
	parsedURL, err := srv.parseShortURL(shortURL)
	if err != nil {
		return nil, err
	}

	s := ShortURL{
		URL:        shortURLKey(parsedURL),
		AccessTime: time.Now(),
	}

	u, err := srv.urlRepository.GetIfNotExpired(ctx, &s, func(lastAccess *time.Time, createdAt time.Time) bool {
		if lastAccess == nil {
			lastAccess = &createdAt
		}
		if lastAccess.Add(srv.expiredAfter).Before(time.Now()) {
			srv.log.Info().Msgf("%s is expired", parsedURL.String())
			return true
		}

		return false
	})

	if err != nil {
		return nil, err
	}

	if err := srv.urlRepository.IncLong(ctx); err != nil {
		srv.log.Err(err).Msg("the attempt to increase the count of 'long' calls")
	}

	return u, nil
}

func (srv *Service) Statistics(ctx context.Context) (*OverallStatistics, error) {
	shortURLStat, err := srv.urlRepository.StatShortURL(ctx)
	if err != nil {
		return nil, err
	}

	longURLStat, err := srv.urlRepository.StatLongURL(ctx)
	if err != nil {
		return nil, err
	}

	return &OverallStatistics{
		LongURL:  *longURLStat,
		ShortURL: *shortURLStat,
	}, nil
}

func (srv *Service) makeShortURL(longURL string) *url.URL {
	salt := strconv.FormatInt(time.Now().Unix(), 10)
	hash := hashWithSalt(longURL, salt)[:shortURLPathLength]
	return &url.URL{
		Scheme: srv.scheme,
		Host:   srv.hostName,
		Path:   hash,
	}

}

func hashWithSalt(str, salt string) string {
	h := md5.New()
	h.Write([]byte(str + salt))
	return hex.EncodeToString(h.Sum(nil))
}

func (srv *Service) parseShortURL(shortURL string) (*url.URL, error) {
	parsedURL, err := url.Parse(shortURL)
	if err != nil {
		return nil, NewBadParamsError("invalid url format", err)
	}

	if parsedURL.Host != srv.hostName || parsedURL.Scheme != srv.scheme {
		return nil, NewBadParamsError("invalid scheme or host name", nil)
	}

	return parsedURL, nil
}

func shortURLKey(url *url.URL) string {
	return strings.TrimLeft(url.Path, "/")
}
