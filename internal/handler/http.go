package handler

import (
	"github.com/kalinink/simple-url-shortener/internal/shortener"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
	"net/http"
)

type HTTPHandler struct {
	e          *echo.Echo
	urlService shortener.URLShortenerService
	log        *zerolog.Logger
}

func NewHTTPHandler(service shortener.URLShortenerService, log *zerolog.Logger) *HTTPHandler {
	e := echo.New()
	e.Debug = false
	e.HideBanner = true
	e.HidePort = true

	h := &HTTPHandler{e: e, urlService: service, log: log}
	h.registerRoutes()

	return h
}

func (hdl *HTTPHandler) RegisterAndStartServer(s *http.Server) error {
	return hdl.e.StartServer(s)
}

func (hdl *HTTPHandler) registerRoutes() {
	hdl.e.Use(middleware.Logger())
	hdl.e.Use(middleware.Recover())

	hdl.e.POST("/short", hdl.createShortURL)
	hdl.e.POST("/long", hdl.getLongURL)
	hdl.e.GET("/statistics", hdl.getStatistics)
}

func (hdl *HTTPHandler) createShortURL(c echo.Context) error {
	longURL := URLRequest{}
	if err := c.Bind(&longURL); err != nil {
		return RespondError(c, err, http.StatusBadRequest)
	}

	url, err := hdl.urlService.CreateShortURL(c.Request().Context(), longURL.URL)
	if err != nil {
		return hdl.handleShortenerServiceError(c, err)
	}

	return Respond(c, URLResponse{url.Short}, http.StatusCreated)
}

func (hdl *HTTPHandler) getLongURL(c echo.Context) error {
	shortURL := URLRequest{}
	if err := c.Bind(&shortURL); err != nil {
		return RespondError(c, err, http.StatusBadRequest)
	}

	url, err := hdl.urlService.GetLongURL(c.Request().Context(), shortURL.URL)
	if err != nil {
		return hdl.handleShortenerServiceError(c, err)
	}

	return Respond(c, URLResponse{url.Short}, http.StatusOK)
}

func (hdl *HTTPHandler) getStatistics(c echo.Context) error {
	stat, err := hdl.urlService.Statistics(c.Request().Context())
	if err != nil {
		return hdl.handleShortenerServiceError(c, err)
	}

	return Respond(c, serviceStatToResponseDTO(stat), http.StatusOK)
}
