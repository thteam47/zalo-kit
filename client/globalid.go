package client

import (
	"errors"
	"fmt"
	"strings"
)

// ErrGlobalIDUnsupported báo bản zago đang dùng chưa có hai hàm gid.
//
// Vì sao phải dò lúc chạy thay vì gọi thẳng: zago đã chuyển private từ
// 27/08/2026, nên go.mod của zalo-kit chỉ ghim được bản cũ mà proxy của Go
// còn giữ — bản đó CHƯA có EncodeGlobalID/DecodeGlobalID. Các sản phẩm dùng
// zalo-kit thay zago bằng bản mới qua replace, và chỉ ở đó hai hàm này mới tồn
// tại. Gọi thẳng thì CI của chính zalo-kit không biên dịch nổi.
var ErrGlobalIDUnsupported = errors.New("zalo-kit: this zago build has no global ID support")

type globalIDAPI interface {
	EncodeGlobalID(userIDs ...string) (map[string]string, error)
	DecodeGlobalID(globalIDs ...string) (map[string]string, error)
}

func (c *Client) globalIDs() (globalIDAPI, error) {
	if c == nil || c.api == nil {
		return nil, ErrSessionInvalid
	}
	api, ok := any(c.api).(globalIDAPI)
	if !ok {
		return nil, ErrGlobalIDUnsupported
	}
	return api, nil
}

/*
EncodeGlobalIDs đổi uid (theo góc nhìn của CHÍNH tài khoản này) ra globalId.

uid Zalo là riêng theo từng tài khoản: cùng một người, nick A thấy một số,
nick B thấy số khác. globalId thì như nhau với mọi nick, nên nó mới là định
danh gốc khi một cửa hàng chạy nhiều nick. Chỉ tài khoản đã thấy uid đó mới
đổi đúng được — đưa uid của nick A cho nick B đổi là ra rác.

Trả về map uid -> globalId. uid nào Zalo không trả thì vắng mặt trong map.
*/
func (c *Client) EncodeGlobalIDs(userIDs ...string) (map[string]string, error) {
	ids := uniqueIDs(userIDs)
	if len(ids) == 0 {
		return map[string]string{}, nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	api, err := c.globalIDs()
	if err != nil {
		return nil, err
	}
	out, err := api.EncodeGlobalID(ids...)
	if err != nil {
		return nil, fmt.Errorf("zalo-kit: encode Zalo global IDs: %w", err)
	}
	return out, nil
}

// DecodeGlobalIDs đổi globalId ra uid mà CHÍNH tài khoản này dùng để nhắn.
// Trả về map globalId -> uid.
func (c *Client) DecodeGlobalIDs(globalIDs ...string) (map[string]string, error) {
	ids := uniqueIDs(globalIDs)
	if len(ids) == 0 {
		return map[string]string{}, nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	api, err := c.globalIDs()
	if err != nil {
		return nil, err
	}
	out, err := api.DecodeGlobalID(ids...)
	if err != nil {
		return nil, fmt.Errorf("zalo-kit: decode Zalo global IDs: %w", err)
	}
	return out, nil
}

func uniqueIDs(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}
