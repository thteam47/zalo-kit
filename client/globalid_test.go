package client

import (
	"errors"
	"reflect"
	"testing"
)

func TestUniqueIDsBoKhoangTrangVaTrung(t *testing.T) {
	got := uniqueIDs([]string{" 1 ", "", "2", "1", "  "})
	if !reflect.DeepEqual(got, []string{"1", "2"}) {
		t.Fatalf("uniqueIDs = %v", got)
	}
}

func TestGlobalIDKhongPhienThiBaoPhienHong(t *testing.T) {
	var c *Client
	if _, err := c.globalIDs(); !errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("err = %v, muốn ErrSessionInvalid", err)
	}
	if got, err := (&Client{}).EncodeGlobalIDs(" ", ""); err != nil || len(got) != 0 {
		t.Fatalf("danh sách rỗng không được gọi Zalo: %v %v", got, err)
	}
}
