package wpsoauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Config struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	AuthURL      string
	TokenURL     string
	UserInfoURL  string
	Scopes       string
	HTTPTimeout  time.Duration
}

type Client struct {
	cfg        Config
	httpClient *http.Client
}

func New(cfg Config) *Client {
	to := cfg.HTTPTimeout
	if to <= 0 {
		to = 10 * time.Second
	}
	return &Client{cfg: cfg, httpClient: &http.Client{Timeout: to}}
}

func (c *Client) AuthCodeURL(state string) string {
	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", c.cfg.ClientID)
	q.Set("redirect_uri", c.cfg.RedirectURI)
	q.Set("scope", c.cfg.Scopes)
	q.Set("state", state)
	// 尽量拉起登录/选帐号页，减少「已登录 SSO 一闪而过」看不出跳转的情况
	q.Set("prompt", "login")
	return c.cfg.AuthURL + "?" + q.Encode()
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

func (c *Client) ExchangeCode(ctx context.Context, code string) (*TokenResponse, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", c.cfg.RedirectURI)
	form.Set("client_id", c.cfg.ClientID)
	form.Set("client_secret", c.cfg.ClientSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("token exchange status=%d body=%s", resp.StatusCode, string(raw))
	}
	var out TokenResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	if out.AccessToken == "" {
		return nil, fmt.Errorf("empty access_token: %s", string(raw))
	}
	return &out, nil
}

type UserInfo struct {
	ID       string
	Nickname string
}

// GetCurrentUser 调用 GET /v7/users/current
func (c *Client) GetCurrentUser(ctx context.Context, accessToken string) (*UserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.cfg.UserInfoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("userinfo status=%d body=%s", resp.StatusCode, string(raw))
	}

	// 兼容多种返回结构
	var envelope map[string]any
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, err
	}
	id, nick := pickUser(envelope)
	if id == "" {
		return nil, fmt.Errorf("cannot parse user id from: %s", string(raw))
	}
	if nick == "" {
		nick = id
	}
	return &UserInfo{ID: id, Nickname: nick}, nil
}

func pickUser(m map[string]any) (id, nick string) {
	if data, ok := m["data"].(map[string]any); ok {
		id = asString(data["id"], data["user_id"], data["userid"], data["uid"])
		nick = asString(data["nickname"], data["name"], data["display_name"])
		return
	}
	id = asString(m["id"], m["user_id"], m["userid"], m["uid"])
	nick = asString(m["nickname"], m["name"], m["display_name"])
	return
}

func asString(vals ...any) string {
	for _, v := range vals {
		switch t := v.(type) {
		case string:
			if t != "" {
				return t
			}
		case float64:
			return fmt.Sprintf("%.0f", t)
		case json.Number:
			return t.String()
		}
	}
	return ""
}
