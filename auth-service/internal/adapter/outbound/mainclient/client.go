package mainclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/sony/gobreaker"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"abet/auth-service/internal/core/domain"
	"abet/auth-service/internal/core/port"
)

// HTTPClient calls the internal API of main-service, wrapped with a gobreaker circuit breaker.
type HTTPClient struct {
	baseURL    string
	token      string
	httpClient *http.Client
	cb         *gobreaker.CircuitBreaker
}

func NewHTTPClient(baseURL, internalToken string, timeout time.Duration) *HTTPClient {
	cbSettings := gobreaker.Settings{
		Name:        "MainServiceClient",
		MaxRequests: 3,
		Interval:    10 * time.Second,
		Timeout:     5 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 5 && failureRatio >= 0.5
		},
	}

	return &HTTPClient{
		baseURL:    baseURL,
		token:      internalToken,
		httpClient: &http.Client{Timeout: timeout, Transport: otelhttp.NewTransport(http.DefaultTransport)},
		cb:         gobreaker.NewCircuitBreaker(cbSettings),
	}
}

type loginSyncRequest struct {
	Email      string `json:"email"`
	Name       string `json:"name"`
	PictureURL string `json:"picture_url"`
}

type loginSyncResponse struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
}

// syncOutcome lets a 403 (a normal business answer) pass through the circuit breaker
// without counting as a failure.
type syncOutcome struct {
	result      port.LoginSyncResult
	notProvided bool
}

// LoginSync asks main-service to update the profile of the user and return its id and role.
// A 403 (user not provisioned) maps to domain.ErrForbidden.
func (c *HTTPClient) LoginSync(ctx context.Context, email, name, pictureURL string) (port.LoginSyncResult, error) {
	res, err := c.cb.Execute(func() (interface{}, error) {
		body, err := json.Marshal(loginSyncRequest{Email: email, Name: name, PictureURL: pictureURL})
		if err != nil {
			return nil, err
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/internal/v1/auth/login-sync", bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Internal-Token", c.token)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("%w: network error: %v", domain.ErrServiceUnavailable, err)
		}
		defer resp.Body.Close()

		switch {
		case resp.StatusCode == http.StatusForbidden:
			return syncOutcome{notProvided: true}, nil
		case resp.StatusCode != http.StatusOK:
			return nil, fmt.Errorf("%w: login-sync status %d", domain.ErrServiceUnavailable, resp.StatusCode)
		}

		var out loginSyncResponse
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			return nil, fmt.Errorf("%w: decode login-sync response: %v", domain.ErrServiceUnavailable, err)
		}
		userID, err := uuid.Parse(out.UserID)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid user_id in login-sync response", domain.ErrServiceUnavailable)
		}
		return syncOutcome{result: port.LoginSyncResult{UserID: userID, Email: out.Email, Role: out.Role}}, nil
	})

	if err != nil {
		if errors.Is(err, gobreaker.ErrOpenState) || errors.Is(err, gobreaker.ErrTooManyRequests) {
			return port.LoginSyncResult{}, domain.ErrCircuitOpen
		}
		return port.LoginSyncResult{}, err
	}

	outcome := res.(syncOutcome)
	if outcome.notProvided {
		return port.LoginSyncResult{}, fmt.Errorf("%w: user is not provisioned", domain.ErrForbidden)
	}
	return outcome.result, nil
}

var _ port.MainServiceClient = (*HTTPClient)(nil)
