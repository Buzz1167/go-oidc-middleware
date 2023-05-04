package shared

import (
	"fmt"
	"net"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func newEchoJWTClaimsHandler[T any](c echo.Context) error {
	claims, ok := c.Get("user").(T)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
	}

	return c.JSON(http.StatusOK, claims)
}

func RunEchoJWT[T any](echoMiddleware echo.MiddlewareFunc, address string, port int) error {
	e := echo.New()
	e.HideBanner = true

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.Secure())
	e.Use(echoMiddleware)

	handler := newEchoJWTClaimsHandler[T]

	e.GET("/", handler)

	addr := net.JoinHostPort(address, fmt.Sprintf("%d", port))
	return e.Start(addr)
}
