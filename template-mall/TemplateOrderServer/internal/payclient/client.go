package payclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func New(baseURL string, timeout time.Duration) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

type CreatePaymentRequest struct {
	OrderID   string `json:"order_id"`
	AmountFen int64  `json:"amount_fen"`
	UserID    string `json:"user_id"`
	Subject   string `json:"subject"`
}

type CreatePaymentResponse struct {
	PayOrderID string `json:"pay_order_id"`
	OrderID    string `json:"order_id"`
	AmountFen  int64  `json:"amount_fen"`
	PayURL     string `json:"pay_url"`
	Status     string `json:"status"`
}

func (c *Client) CreatePayment(ctx context.Context, req CreatePaymentRequest) (*CreatePaymentResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/payments", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("create payment: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("create payment status=%d body=%s", resp.StatusCode, string(raw))
	}
	var out CreatePaymentResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("decode payment response: %w", err)
	}
	return &out, nil
}
