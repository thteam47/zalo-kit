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
	Scanned bool   `json:"scanned"`
	Name    string `json:"name,omitempty"`
	UserID  string `json:"userId,omitempty"`
	Phone   string `json:"phone,omitempty"`
	Avatar  string `json:"avatar,omitempty"`
	// Profile giữ nguyên phản hồi thô để bên gọi cần gì thì tự đọc thêm.
	Profile map[string]any `json:"profile,omitempty"`
}

// QRSession là phiên thu được sau khi người dùng bấm xác nhận trên điện thoại.
type QRSession struct {
	Cookies     map[string]string
	UserID      string
	DisplayName string
	Phone       string
	Avatar      string
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
	scanned   QRScan
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
	scan := QRScan{Scanned: responseCode(profile) == 0, Profile: profile}
	if scan.Scanned {
		// Danh tính chỉ xuất hiện ở đúng lần quét này. Sau khi xác nhận, Zalo
		// thường KHÔNG trả lại tên/uid nữa, nên phải giữ lại ngay.
		payload := unwrapPayload(profile)
		scan.Name = firstString(payload, "display_name", "displayName", "name")
		scan.Avatar = firstString(payload, "avatar", "avatarUrl", "avatar_url", "avt")
		scan.Phone = firstString(payload, "phone", "phone_number", "phoneNumber")
		scan.UserID = firstID(payload, "uid", "userId", "user_id", "owner_id", "ownerId")
		c.scanned = scan
	}
	return scan, nil
}

// unwrapPayload gỡ lớp bọc data/profile của phản hồi Zalo.
func unwrapPayload(raw map[string]any) map[string]any {
	if raw == nil {
		return nil
	}
	for _, key := range []string{"data", "Data", "profile", "Profile"} {
		if nested, ok := raw[key].(map[string]any); ok && nested != nil {
			return nested
		}
	}
	return raw
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
	session := QRSession{
		Cookies:     cookies,
		UserID:      strings.TrimSpace(c.api.UID()),
		DisplayName: strings.TrimSpace(c.api.AccountName()),
		Phone:       c.scanned.Phone,
		Avatar:      c.scanned.Avatar,
	}
	// Sau khi xác nhận, hai trường này hay rỗng; lấy lại từ lúc quét.
	if session.UserID == "" {
		session.UserID = c.scanned.UserID
	}
	if session.DisplayName == "" {
		session.DisplayName = c.scanned.Name
	}
	return session, nil
}

// Event là một sự kiện ngoài tin nhắn: đã nhận, đã xem, biến động trong nhóm,
// lỗi đường truyền, hoặc tải tệp xong.
//
// Một kiểu chung cho cả năm loại thay vì năm channel: nơi gọi chỉ phải nối MỘT
// chỗ, và thêm loại mới không bắt nó sửa chữ ký hàm.
type Event struct {
	Kind      EventKind
	ThreadID  string
	MessageID string
	At        time.Time

	// GroupEventType chỉ có ở EventGroup: "join", "leave", "remove_member",
	// "add_admin"… Để nguyên văn chuỗi của Zalo chứ không dựng enum riêng —
	// Zalo thêm loại mới bất cứ lúc nào, và enum của ta thiếu một giá trị thì
	// sự kiện đó thành "unknown" rồi bị bỏ im lặng.
	GroupEventType string
	// Data là nội dung thô của sự kiện nhóm.
	Data map[string]any

	// Err chỉ có ở EventSocketError.
	Err error

	// FileID/FileURL chỉ có ở EventUploadDone.
	FileID  string
	FileURL string
}

type EventKind string

const (
	EventDelivered   EventKind = "delivered"
	EventSeen        EventKind = "seen"
	EventGroup       EventKind = "group"
	EventSocketError EventKind = "socket_error"
	EventUploadDone  EventKind = "upload_done"
)

func (c *Client) Listen(ctx context.Context, onMessage func(inbound.Message), onError func(error)) error {
	return c.ListenWithEvents(ctx, onMessage, nil, onError)
}

// ListenWithEvents giống Listen nhưng nhận thêm luồng đã-nhận / đã-xem.
//
// Giữ Listen cũ nguyên chữ ký: nơi gọi không cần sự kiện thì không phải sửa,
// và truyền nil có nghĩa là KHÔNG rút channel — xem cảnh báo ở dưới.
func (c *Client) ListenWithEvents(ctx context.Context, onMessage func(inbound.Message),
	onEvent func(Event), onError func(error)) error {
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
	// Luồng sự kiện đã-nhận / đã-xem.
	//
	// ⚠️ Trước đây chỉ đăng ký bộ lắng nghe TIN NHẮN, nên hai kênh này bị bỏ
	// trống hoàn toàn: hệ thống không bao giờ biết tin đã tới máy khách hay khách
	// đã xem. Đây là channel chứ không phải callback, nên phải có vòng rút — không
	// rút thì zago đầy channel rồi nghẽn chính vòng nhận tin.
	if onEvent != nil {
		go c.runEvents(ctx, onEvent)
	}
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

// runEvents rút hai channel sự kiện của zago cho tới khi ngắt.
//
// ⚠️ Zalo gửi MẢNG mã tin trong một sự kiện (MsgIDs là `any`, thực tế là mảng
// hoặc một chuỗi). Chuẩn hoá tại đây thành từng sự kiện một để bên gọi không
// phải đoán lại hình dạng.
func (c *Client) runEvents(ctx context.Context, onEvent func(Event)) {
	delivered := c.api.DeliveryEvents()
	seen := c.api.SeenEvents()
	groups := c.api.GroupEvents()
	socketErrs := c.api.SocketErrors()
	uploads := c.api.UploadEvents()
	for {
		select {
		case <-ctx.Done():
			return
		case evt, ok := <-delivered:
			if !ok {
				delivered = nil
				continue
			}
			phatSuKien(onEvent, EventDelivered, evt.ThreadID, evt.MsgIDs, evt.Timestamp)
		case evt, ok := <-seen:
			if !ok {
				seen = nil
				continue
			}
			phatSuKien(onEvent, EventSeen, evt.ThreadID, evt.MsgIDs, evt.Timestamp)
		case evt, ok := <-groups:
			if !ok {
				groups = nil
				continue
			}
			onEvent(suKienNhom(evt))
		case evt, ok := <-socketErrs:
			if !ok {
				socketErrs = nil
				continue
			}
			onEvent(Event{Kind: EventSocketError, Err: evt.Err, At: mocThoiGian(evt.Timestamp)})
		case evt, ok := <-uploads:
			if !ok {
				uploads = nil
				continue
			}
			onEvent(Event{Kind: EventUploadDone, FileID: evt.FileID,
				FileURL: evt.FileURL, At: time.Now().UTC()})
		}
	}
}

// suKienNhom đổi phong bì sự kiện nhóm của zago sang dạng của ta.
//
// ⚠️ evt.Event là CON TRỎ và zago có nhánh gửi nil. Gọi ToMap() trên nil là
// panic, mà panic ở đây giết luôn vòng rút channel — mọi sự kiện sau đó biến
// mất im lặng.
func suKienNhom(evt zago.GroupEventEnvelope) Event {
	out := Event{
		Kind:           EventGroup,
		GroupEventType: string(evt.EventType),
		At:             time.Now().UTC(),
	}
	if evt.Event == nil {
		return out
	}
	out.Data = evt.Event.ToMap()
	out.ThreadID = firstID(out.Data, "groupId", "grid", "threadId", "group_id")
	out.MessageID = firstID(out.Data, "msgId", "globalMsgId")
	return out
}

func mocThoiGian(ts int64) time.Time {
	if ts <= 0 {
		return time.Now().UTC()
	}
	if ts > 1_000_000_000_000 {
		return time.UnixMilli(ts).UTC()
	}
	return time.Unix(ts, 0).UTC()
}

func phatSuKien(onEvent func(Event), kind EventKind, threadID string, msgIDs any, ts int64) {
	at := mocThoiGian(ts)
	for _, id := range tachMaTin(msgIDs) {
		onEvent(Event{Kind: kind, ThreadID: cleanID(threadID), MessageID: id, At: at})
	}
}

// tachMaTin đọc MsgIDs ở mọi dạng Zalo từng trả: một chuỗi, một số, hoặc mảng.
func tachMaTin(raw any) []string {
	switch value := raw.(type) {
	case nil:
		return nil
	case string:
		if id := cleanID(value); id != "" {
			return []string{id}
		}
	case []string:
		out := make([]string, 0, len(value))
		for _, item := range value {
			if id := cleanID(item); id != "" {
				out = append(out, id)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(value))
		for _, item := range value {
			out = append(out, tachMaTin(item)...)
		}
		return out
	default:
		if id := cleanID(fmt.Sprintf("%v", value)); id != "" {
			return []string{id}
		}
	}
	return nil
}
