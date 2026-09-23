package client

import (
	"testing"
	"time"
)

// Zalo trả mã tin ở ba dạng khác nhau trong cùng một luồng sự kiện. Đọc sai
// dạng nào thì sự kiện đó biến mất KHÔNG BÁO LỖI — tin vẫn hiện "đang gửi"
// mãi mãi dù khách đã nhận.
func TestTachMaTinChiuDuocMoiDang(t *testing.T) {
	cases := map[string]struct {
		raw  any
		muon []string
	}{
		"một chuỗi":     {"m-1", []string{"m-1"}},
		"mảng chuỗi":    {[]string{"m-1", "m-2"}, []string{"m-1", "m-2"}},
		"mảng any":      {[]any{"m-1", float64(2)}, []string{"m-1", "2"}},
		"số trần":       {float64(7), []string{"7"}},
		"rỗng":          {nil, nil},
		"chuỗi trắng":   {"   ", nil},
		"mảng lẫn rỗng": {[]any{"m-1", "", nil}, []string{"m-1"}},
	}
	for ten, tc := range cases {
		got := tachMaTin(tc.raw)
		if len(got) != len(tc.muon) {
			t.Errorf("%s: nhận %#v, muốn %#v", ten, got, tc.muon)
			continue
		}
		for i := range got {
			if got[i] != tc.muon[i] {
				t.Errorf("%s: nhận %#v, muốn %#v", ten, got, tc.muon)
				break
			}
		}
	}
}

// Một sự kiện mang NHIỀU mã tin phải nở ra thành nhiều sự kiện, nếu không thì
// chỉ tin đầu tiên trong lô được đánh dấu.
func TestMotSuKienNhieuMaThiNoRa(t *testing.T) {
	var got []Event
	phatSuKien(func(e Event) { got = append(got, e) }, EventSeen, "t-1",
		[]any{"m-1", "m-2", "m-3"}, 1_700_000_000_000)
	if len(got) != 3 {
		t.Fatalf("muốn 3 sự kiện, nhận %d", len(got))
	}
	if got[0].ThreadID != "t-1" || got[0].Kind != EventSeen {
		t.Fatalf("sai luồng hoặc loại: %#v", got[0])
	}
	if !got[0].At.Equal(time.UnixMilli(1_700_000_000_000).UTC()) {
		t.Fatalf("mốc thời gian sai: %v", got[0].At)
	}
}

// Zalo gửi mốc thời gian lúc là giây, lúc là mili. Đọc giây thành mili đẩy sự
// kiện về năm 1970; đọc mili thành giây đẩy nó sang năm 55000.
func TestMocThoiGianDoanDungDonVi(t *testing.T) {
	var giay, mili Event
	phatSuKien(func(e Event) { giay = e }, EventDelivered, "t", "m", 1_700_000_000)
	phatSuKien(func(e Event) { mili = e }, EventDelivered, "t", "m", 1_700_000_000_000)
	if !giay.At.Equal(mili.At) {
		t.Fatalf("giây %v khác mili %v", giay.At, mili.At)
	}
	if giay.At.Year() != 2023 {
		t.Fatalf("năm sai: %v", giay.At)
	}
}

// Không có mốc thời gian thì dùng giờ hiện tại, không để zero-time — zero-time
// lọt xuống kho là một tin "gửi năm 0001", nằm trên đầu mọi danh sách.
func TestKhongCoMocThiDungGioHienTai(t *testing.T) {
	var got Event
	phatSuKien(func(e Event) { got = e }, EventSeen, "t", "m", 0)
	if got.At.IsZero() || time.Since(got.At) > time.Minute {
		t.Fatalf("mốc thời gian không hợp lý: %v", got.At)
	}
}
