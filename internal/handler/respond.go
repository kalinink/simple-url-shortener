package handler

import (
	"github.com/kalinink/simple-url-shortener/internal/shortener"
	"github.com/labstack/echo/v4"
	"net/http"
	"time"
)

func RespondError(c echo.Context, err error, status int) error {
	if echoErr, ok := err.(*echo.HTTPError); ok {
		err = echoErr.Internal
	}

	return respond(c, ErrorResponse{err.Error()}, status)
}

func Respond(c echo.Context, data interface{}, status int) error {
	return respond(c, data, status)
}

func respond(c echo.Context, data interface{}, status int) error {
	if status == http.StatusNoContent {
		return nil
	}
	return c.JSON(status, data)
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func RespondInternalError(c echo.Context) error {
	return respond(c, nil, http.StatusInternalServerError)
}

func (hdl *HTTPHandler) handleShortenerServiceError(c echo.Context, err error) error {
	sErr, ok := err.(shortener.Error)
	if !ok {
		hdl.log.Err(err).Msg("")
		return RespondInternalError(c)
	}

	hdl.log.Err(sErr.Origin).Msg("")
	return RespondError(c, sErr, serviceErrorToHTTPError[sErr.Type])
}

var serviceErrorToHTTPError = map[int]int{
	shortener.NotFoundErrType:  http.StatusNotFound,
	shortener.BadParamsErrType: http.StatusBadRequest,
}

type URLResponse struct {
	URL string `json:"url"`
}

type URLRequest struct {
	URL string `json:"url"`
}

type StatisticResponse struct {
	Counts  CountStatistics  `json:"counts"`
	Timings TimingStatistics `json:"timings"`
}

type CountStatistics struct {
	Long  int `json:"long"`
	Short int `json:"short"`
}

type TimingStatistics struct {
	Long  *time.Time `json:"long"`
	Short *time.Time `json:"short"`
}

func serviceStatToResponseDTO(s *shortener.OverallStatistics) *StatisticResponse {
	return &StatisticResponse{
		Counts: CountStatistics{
			Long:  s.LongURL.Count,
			Short: s.ShortURL.Count,
		},
		Timings: TimingStatistics{
			Long:  s.LongURL.Timing,
			Short: s.ShortURL.Timing,
		},
	}
}
