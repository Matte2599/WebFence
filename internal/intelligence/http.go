package intelligence

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"time"
)

func sourceGET(ctx context.Context, client *http.Client, req *http.Request, limit int) ([]byte, error) {
	owned := *client
	owned.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	if owned.Timeout == 0 {
		owned.Timeout = 20 * time.Second
	}
	for attempt := 0; attempt < 3; attempt++ {
		resp, err := owned.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			return nil, ErrUnavailable
		}
		if resp.StatusCode == http.StatusOK {
			data, readErr := io.ReadAll(io.LimitReader(resp.Body, int64(limit)+1))
			_ = resp.Body.Close()
			if readErr != nil {
				return nil, ErrUnavailable
			}
			if len(data) > limit {
				return nil, ErrLimit
			}
			return data, nil
		}
		wait := time.Duration(attempt+1) * time.Second
		if h := resp.Header.Get("Retry-After"); h != "" {
			if seconds, e := strconv.Atoi(h); e == nil && seconds >= 0 {
				wait = time.Duration(seconds) * time.Second
			} else if t, e := http.ParseTime(h); e == nil {
				wait = time.Until(t)
				if wait < 0 {
					wait = 0
				}
			} else {
				_ = resp.Body.Close()
				return nil, ErrUnavailable
			}
		}
		status := resp.StatusCode
		_ = resp.Body.Close()
		if attempt == 2 || (status != http.StatusTooManyRequests && status != http.StatusServiceUnavailable && status != http.StatusBadGateway && status != http.StatusGatewayTimeout) || wait > 30*time.Second {
			return nil, ErrUnavailable
		}
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
	return nil, ErrUnavailable
}
