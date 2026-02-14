package mqtt

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var ErrEMQXEndpointRequired = errors.New("realtime/mqtt: emqx endpoint is required")

type EMQXSessionControllerConfig struct {
	Endpoint    string
	Username    string
	Password    string
	BearerToken string
	Timeout     time.Duration
	TLSConfig   *tls.Config
}

type EMQXSessionController struct {
	baseURL     string
	username    string
	password    string
	bearerToken string
	httpClient  *http.Client
}

func NewEMQXSessionController(cfg EMQXSessionControllerConfig) (*EMQXSessionController, error) {
	baseURL := strings.TrimSpace(cfg.Endpoint)
	if baseURL == "" {
		return nil, ErrEMQXEndpointRequired
	}
	baseURL = strings.TrimRight(baseURL, "/")

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	transport := &http.Transport{}
	if cfg.TLSConfig != nil {
		transport.TLSClientConfig = cfg.TLSConfig
	}

	return &EMQXSessionController{
		baseURL:     baseURL,
		username:    cfg.Username,
		password:    cfg.Password,
		bearerToken: cfg.BearerToken,
		httpClient: &http.Client{
			Timeout:   timeout,
			Transport: transport,
		},
	}, nil
}

func (c *EMQXSessionController) DisconnectClient(ctx context.Context, clientID string, _ string) error {
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		return errors.New("realtime/mqtt: client id is required")
	}

	u := c.baseURL + "/api/v5/clients/" + url.PathEscape(clientID) + "/kickout"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, nil)
	if err != nil {
		return err
	}

	if token := strings.TrimSpace(c.bearerToken); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	} else if c.username != "" || c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusAccepted {
		return nil
	}

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	return fmt.Errorf("realtime/mqtt: emqx kickout failed status=%d body=%q", resp.StatusCode, string(body))
}
