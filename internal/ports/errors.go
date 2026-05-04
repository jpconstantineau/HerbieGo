package ports

import "errors"

var (
	ErrNonResponsive   = errors.New("player did not respond")
	ErrProviderFailure = errors.New("ai provider request failed")
	ErrProviderTimeout = errors.New("ai provider request timed out")
)
