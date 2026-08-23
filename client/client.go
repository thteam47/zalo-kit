package client

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/thteam47/zago"
	"github.com/thteam47/zalo-kit/inbound"
)

var ErrNoQRChallenge = errors.New("zalo-kit: no active QR challenge")

type Options struct {
	AccountID string
	IMEI      string
	UserAgent string
	ProxyURL  string
	Cookies   map[string]string
}

type QRChallenge struct {
	ImageDataURL string    `json:"imageDataUrl"`
	IssuedAt     time.Time `json:"issuedAt"`
	raw          *zago.QRAuthResult
}

type QRScan struct {
	Scanned bool           `json:"scanned"`
	Profile map[string]any `json:"profile,omitempty"`
}

type SendResult struct {
	MessageID       string
	ClientMessageID string
	Raw             any
}

type Client struct {
	api       *zago.ZaloAPI
	accountID string
	mu        sync.Mutex
	qr        *QRChallenge
}

func New(opts Options) (*Client, error) {
	if strings.TrimSpace(opts.AccountID) == "" {
		return nil, errors.New("zalo-kit: account ID is required")
	}
	if strings.TrimSpace(opts.IMEI) == "" {
		return nil, errors.New("zalo-kit: IMEI is required")
	}
	api, err := zago.Zalo("", "", opts.IMEI, opts.Cookies, opts.UserAgent, false, zago.LoginAPI)
	if err != nil {
		return nil, fmt.Errorf("create Zalo client: %w", err)
	}
	if opts.ProxyURL != "" {
		if err := api.SetProxyURL(opts.ProxyURL); err != nil {
			return nil, fmt.Errorf("set Zalo proxy: %w", err)
		}
	}
	return &Client{api: api, accountID: opts.AccountID}, nil
}

func (c *Client) IsLoggedIn() bool { return c != nil && c.api != nil && c.api.IsLoggedIn() }

func (c *Client) SetSession(cookies map[string]string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.api.SetSession(cookies)
}

func (c *Client) GenerateQR() (QRChallenge, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	raw, err := c.api.AuthQRCode()
	if err != nil {
		return QRChallenge{}, fmt.Errorf("generate Zalo QR: %w", err)
	}
	challenge := QRChallenge{
		ImageDataURL: "data:image/png;base64," + base64.StdEncoding.EncodeToString(raw.ImageBytes),
		IssuedAt:     time.Now().UTC(),
		raw:          raw,
	}
	c.qr = &challenge
	return challenge, nil
}

func (c *Client) CheckQRScan() (QRScan, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.qr == nil || c.qr.raw == nil {
		return QRScan{}, ErrNoQRChallenge
	}
	profile, err := c.api.CheckQRCodeScan(c.qr.raw)
	if err != nil {
		return QRScan{}, fmt.Errorf("check Zalo QR scan: %w", err)
	}
	return QRScan{Scanned: responseCode(profile) == 0, Profile: profile}, nil
}

func (c *Client) WaitQRConfirm(ctx context.Context, interval time.Duration) (map[string]string, error) {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		c.mu.Lock()
		if c.qr == nil || c.qr.raw == nil {
			c.mu.Unlock()
			return nil, ErrNoQRChallenge
		}
		result, err := c.api.CheckQRCodeConfirm(c.qr.raw)
		if err == nil && responseCode(result) == 0 {
			cookies, cookieErr := c.api.QRCookies(c.qr.raw)
			c.mu.Unlock()
			if cookieErr != nil {
				return nil, fmt.Errorf("read Zalo QR session: %w", cookieErr)
			}
			return cookies, nil
		}
		c.mu.Unlock()
		if err != nil {
			return nil, fmt.Errorf("confirm Zalo QR: %w", err)
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}

func (c *Client) Listen(ctx context.Context, onMessage func(inbound.Message), onError func(error)) error {
	c.api.SetMessageListener(func(mid, userID, text string, data *zago.MessageObject, threadID string, tt zago.ThreadType) {
		if onMessage != nil {
			onMessage(normalizeMessage(c.accountID, c.api.UserID(), mid, userID, text, data, threadID, tt, time.Now().UTC()))
		}
	})
	c.api.SetErrorListener(func(err error, _ int64) {
		if err != nil && onError != nil {
			onError(err)
		}
	})
	done := make(chan error, 1)
	go func() { done <- c.api.Listen(false, 0) }()
	select {
	case <-ctx.Done():
		_ = c.api.StopListening()
		return ctx.Err()
	case err := <-done:
		return err
	}
}

func (c *Client) SendText(_ context.Context, threadID string, threadType inbound.ThreadType, text string) (SendResult, error) {
	if strings.TrimSpace(threadID) == "" || strings.TrimSpace(text) == "" {
		return SendResult{}, errors.New("zalo-kit: thread ID and text are required")
	}
	tt := zago.ThreadTypeUSER
	if threadType == inbound.ThreadGroup {
		tt = zago.ThreadTypeGROUP
	}
	c.mu.Lock()
	raw, err := c.api.SendMessage(zago.Message{Text: text}, threadID, tt)
	c.mu.Unlock()
	if err != nil {
		return SendResult{}, fmt.Errorf("send Zalo message: %w", err)
	}
	ids := extractIDs(raw)
	return SendResult{MessageID: ids["msgId"], ClientMessageID: ids["cliMsgId"], Raw: raw}, nil
}
