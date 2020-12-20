package handler

import (
	"github.com/kalinink/simple-url-shortener/internal/shortener"
	"github.com/labstack/echo/v4"
	"net/http"
	"time"
)

const layout = "2006-01-02 15:04:05"

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
} // @name Error

func RespondInternalError(c echo.Context) error {
	return respond(c, ErrorResponse{"internal server error"}, http.StatusInternalServerError)
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
} // @name Response

type URLRequest struct {
	URL string `json:"url"`
} // @name Request

type StatisticResponse struct {
	Counts  CountStatistics  `json:"counts"`
	Timings TimingStatistics `json:"timings"`
} // @name Statistics

type CountStatistics struct {
	Long  int `json:"long" example:"10"`
	Short int `json:"short" example:"5"`
} // @name CountStatistics

type TimingStatistics struct {
	Long  string `json:"long" example:"2020-11-10 12:00:05"`
	Short string `json:"short" example:"2020-10-23 01:33:45"`
} // @name TimingStatistics

func serviceStatToResponseDTO(s *shortener.OverallStatistics) *StatisticResponse {
	return &StatisticResponse{
		Counts: CountStatistics{
			Long:  s.LongURL.Count,
			Short: s.ShortURL.Count,
		},
		Timings: TimingStatistics{
			Long:  formatTime(s.LongURL.Timing, layout),
			Short: formatTime(s.ShortURL.Timing, layout),
		},
	}
}

func formatTime(t *time.Time, layout string) string {
	if t == nil {
		return ""
	}
	return t.Format(layout)
}
