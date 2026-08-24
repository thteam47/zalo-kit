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
	"github.com/thteam47/zalo-kit/health"
	"github.com/thteam47/zalo-kit/inbound"
)

var (
	ErrNoQRChallenge  = errors.New("zalo-kit: no active QR challenge")
	ErrSessionInvalid = errors.New("zalo-kit: Zalo session is invalid or expired")
)

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

// QRSession là phiên thu được sau khi người dùng bấm xác nhận trên điện thoại.
type QRSession struct {
	Cookies     map[string]string
	UserID      string
	DisplayName string
}

type SendResult struct {
	MessageID       string
	ClientMessageID string
	Raw             any
}

type Client struct {
	api       *zago.ZaloAPI
	accountID string
	imei      string
	userAgent string
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
	client := &Client{api: api, accountID: opts.AccountID, imei: opts.IMEI, userAgent: opts.UserAgent}
	if len(opts.Cookies) > 0 {
		api.SetSession(opts.Cookies)
		// Cookie còn sống thì dùng luôn. Gọi Login khi không cần chỉ tổ làm
		// Zalo cấp lại khoá phiên và vứt cái đang dùng được.
		if !api.IsLoggedIn() {
			if err := client.hydrateSession(); err != nil {
				return nil, err
			}
		}
	}
	return client, nil
}

func (c *Client) IsLoggedIn() bool { return c != nil && c.api != nil && c.api.IsLoggedIn() }

func (c *Client) SetSession(cookies map[string]string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.api.SetSession(cookies) {
		return false
	}
	return c.hydrateSession() == nil
}

func (c *Client) hydrateSession() error {
	if err := c.api.Login("", "", c.imei, c.userAgent); err != nil {
		if health.Classify(err) == health.FailureAuth {
			return fmt.Errorf("%w: %v", ErrSessionInvalid, err)
		}
		return fmt.Errorf("hydrate Zalo session: %w", err)
	}
	return nil
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

func (c *Client) WaitQRConfirm(ctx context.Context, interval time.Duration) (QRSession, error) {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	for {
		select {
		case <-ctx.Done():
			return QRSession{}, ctx.Err()
		default:
		}
		c.mu.Lock()
		if c.qr == nil || c.qr.raw == nil {
			c.mu.Unlock()
			return QRSession{}, ErrNoQRChallenge
		}
		result, err := c.api.CheckQRCodeConfirm(c.qr.raw)
		if err == nil && responseCode(result) == 0 {
			session, sessionErr := c.finishQR()
			c.mu.Unlock()
			return session, sessionErr
		}
		c.mu.Unlock()
		if err != nil {
			return QRSession{}, fmt.Errorf("confirm Zalo QR: %w", err)
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return QRSession{}, ctx.Err()
		case <-timer.C:
		}
	}
}

// finishQR chốt phiên sau khi người dùng đã xác nhận. Hai lời gọi CheckQRSession
// và FetchQRUserInfo là bắt buộc: bỏ qua chúng thì cookie đọc được vẫn có nhưng
// chưa thành phiên thật, và lần đăng nhập sau sẽ trả lỗi #102 "session key was
// improperly submitted" — trông hệt như hết phiên, rất dễ chẩn đoán nhầm.
// Người gọi đang giữ khoá.
func (c *Client) finishQR() (QRSession, error) {
	if _, err := c.api.CheckQRSession(c.qr.raw); err != nil {
		return QRSession{}, fmt.Errorf("finalize Zalo QR session: %w", err)
	}
	if _, err := c.api.FetchQRUserInfo(c.qr.raw); err != nil {
		return QRSession{}, fmt.Errorf("read Zalo account info: %w", err)
	}
	cookies, err := c.api.QRCookies(c.qr.raw)
	if err != nil {
		return QRSession{}, fmt.Errorf("read Zalo QR session: %w", err)
	}
	if len(cookies) == 0 {
		return QRSession{}, errors.New("zalo-kit: Zalo returned no session cookies after confirmation")
	}
	return QRSession{
		Cookies:     cookies,
		UserID:      c.api.UID(),
		DisplayName: strings.TrimSpace(c.api.AccountName()),
	}, nil
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

// SetTyping bật chỉ báo "đang soạn tin" của Zalo. Trả lời tức thì là dấu hiệu
// máy rõ nhất, nên sản phẩm dùng chỉ báo này cùng với độ trễ theo độ dài tin.
func (c *Client) SetTyping(_ context.Context, threadID string, threadType inbound.ThreadType) error {
	if strings.TrimSpace(threadID) == "" {
		return errors.New("zalo-kit: thread ID is required")
	}
	c.mu.Lock()
	_, err := c.api.SetTyping(threadID, zaloThreadType(threadType))
	c.mu.Unlock()
	if err != nil {
		return fmt.Errorf("set Zalo typing: %w", err)
	}
	return nil
}

func zaloThreadType(threadType inbound.ThreadType) zago.ThreadType {
	if threadType == inbound.ThreadGroup {
		return zago.ThreadTypeGROUP
	}
	return zago.ThreadTypeUSER
}

func (c *Client) SendText(_ context.Context, threadID string, threadType inbound.ThreadType, text string) (SendResult, error) {
	if strings.TrimSpace(threadID) == "" || strings.TrimSpace(text) == "" {
		return SendResult{}, errors.New("zalo-kit: thread ID and text are required")
	}
	c.mu.Lock()
	raw, err := c.api.SendMessage(zago.Message{Text: text}, threadID, zaloThreadType(threadType))
	c.mu.Unlock()
	if err != nil {
		return SendResult{}, fmt.Errorf("send Zalo message: %w", err)
	}
	ids := extractIDs(raw)
	return SendResult{MessageID: ids["msgId"], ClientMessageID: ids["cliMsgId"], Raw: raw}, nil
}
