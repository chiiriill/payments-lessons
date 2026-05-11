package mockclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"stepik-payments-course/internal/usecase/checkPaymentUseCase"
	"stepik-payments-course/internal/usecase/createPaymentUseCase"
	"stepik-payments-course/internal/usecase/processStalePaymentsUseCase"
	"stepik-payments-course/internal/usecase/refundPaymentUseCase"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
	retries    int
}

func New(baseURL string, timeout time.Duration, retries int) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: timeout},
		retries:    retries,
	}
}

func (c *Client) CreatePayment(ctx context.Context, req createPaymentUseCase.ProviderCreateRequest) (createPaymentUseCase.ProviderCreateResponse, error) {
	var res createPaymentUseCase.ProviderCreateResponse
	err := c.do(ctx, http.MethodPost, "/payments", req, &res)
	return res, err
}

func (c *Client) CheckPayment(ctx context.Context, providerPaymentID string) (checkPaymentUseCase.ProviderStatusResponse, error) {
	var res checkPaymentUseCase.ProviderStatusResponse
	err := c.do(ctx, http.MethodGet, "/payments/"+providerPaymentID, nil, &res)
	return res, err
}

func (c *Client) CheckPaymentForWorker(ctx context.Context, providerPaymentID string) (processStalePaymentsUseCase.ProviderStatusResponse, error) {
	var res processStalePaymentsUseCase.ProviderStatusResponse
	err := c.do(ctx, http.MethodGet, "/payments/"+providerPaymentID, nil, &res)
	return res, err
}

func (c *Client) RefundPayment(ctx context.Context, req refundPaymentUseCase.ProviderRefundRequest) error {
	body := map[string]any{"amount": req.Amount}
	return c.do(ctx, http.MethodPost, "/payments/"+req.ProviderPaymentID+"/refund", body, nil)
}

// permanentError wraps a 4xx status code — these are never retried.
type permanentError struct{ statusCode int }

func (e *permanentError) Error() string {
	return fmt.Sprintf("provider rejected request: status %d", e.statusCode)
}

func (c *Client) do(ctx context.Context, method string, path string, body any, out any) error {
	var lastErr error
	attempts := c.retries + 1
	for attempt := 0; attempt < attempts; attempt++ {
		err := c.doOnce(ctx, method, path, body, out)
		if err == nil {
			return nil
		}
		lastErr = err
		if ctx.Err() != nil {
			return ctx.Err()
		}
		var pErr *permanentError
		if errors.As(err, &pErr) {
			return err
		}
		time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
	}
	return lastErr
}

func (c *Client) doOnce(ctx context.Context, method string, path string, body any, out any) error {
	var payload bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&payload).Encode(body); err != nil {
			return err
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, &payload)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode >= http.StatusInternalServerError {
		return fmt.Errorf("provider temporary error: status %d", res.StatusCode)
	}
	if res.StatusCode >= http.StatusBadRequest {
		return &permanentError{statusCode: res.StatusCode}
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(res.Body).Decode(out)
}
