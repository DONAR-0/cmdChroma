package internal

import "log/slog"

func CheckDefer(closeFunc func() error) {
	if err := closeFunc(); err != nil {
		slog.Debug("error received", "err", err)
	}
}
