package comm

import "errors"

var (
	ErrConfigNotLoaded = errors.New("config not loaded, please call LoadConfig() first")
)
