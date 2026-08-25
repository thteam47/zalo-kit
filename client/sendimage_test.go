package client

import (
	"context"
	"strings"
	"testing"

	"github.com/thteam47/zalo-kit/inbound"
)

// Zalo tu di lay anh tu URL, nen duong dan noi bo hay du lieu nhung deu lam
// khach thay anh vo — chan ngay tai cho thay vi de Zalo tra loi kho hieu.
func TestSendImageRefusesUnreachableURL(t *testing.T) {
	client := &Client{}
	for _, url := range []string{"", "/uploads/anh.jpg", "data:image/png;base64,AAAA", "s3://bucket/anh.jpg"} {
		if _, err := client.SendImage(context.Background(), "u-1", inbound.ThreadDirect, url, "", 100, 100); err == nil {
			t.Fatalf("phai tu choi URL khong cong khai: %q", url)
		}
	}
}

func TestSendImageRequiresThread(t *testing.T) {
	client := &Client{}
	_, err := client.SendImage(context.Background(), "  ", inbound.ThreadDirect, "https://s3.zchot.io.vn/a.jpg", "", 10, 10)
	if err == nil || !strings.Contains(err.Error(), "thread ID") {
		t.Fatalf("phai doi thread ID, nhan %v", err)
	}
}
