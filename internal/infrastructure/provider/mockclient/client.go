package mockclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"stepik-payments-course/internal/common/apperror"
	"stepik-payments-course/internal/usecase/checkPaymentUseCase"
	"stepik-payments-course/internal/usecase/createPaymentUseCase"
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

func (c *Client) RefundPayment(ctx context.Context, req refundPaymentUseCase.ProviderRefundRequest) error {
	body := map[string]any{"amount": req.Amount}
	return c.do(ctx, http.MethodPost, "/payments/"+req.ProviderPaymentID+"/refund", body, nil)
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
		if !errors.Is(err, apperror.ErrProviderTemporary) {
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
		return fmt.Errorf("%w: %w", apperror.ErrProviderTemporary, err)
	}
	defer res.Body.Close()

	if res.StatusCode >= http.StatusInternalServerError {
		return fmt.Errorf("%w: status %d", apperror.ErrProviderTemporary, res.StatusCode)
	}
	if res.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("%w: status %d", apperror.ErrProviderPermanent, res.StatusCode)
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(res.Body).Decode(out)
}
