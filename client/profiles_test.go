package client

import "testing"

// Zalo trả hồ sơ lồng nhiều tầng và đổi hình dạng theo phiên bản endpoint, nên
// hàm đọc phải nhặt được ở bất kỳ tầng nào thay vì bám vào một dạng.
func TestCollectProfilesWalksNestedShapes(t *testing.T) {
	raw := map[string]any{
		"changed_profiles": map[string]any{
			"843728390653970339_0": map[string]any{
				"userId": "843728390653970339", "zaloName": "Chị Lan",
				"avatar": "https://cdn/lan.jpg", "phoneNumber": "0901234567",
			},
		},
		"unchanged_profiles": []any{
			map[string]any{"uid": "5845665227953141513", "displayName": "Anh Nam"},
		},
	}
	out := map[string]Profile{}
	collectProfiles(raw, out, 0)
	if len(out) != 2 {
		t.Fatalf("phải đọc được cả hai hồ sơ, nhận %d", len(out))
	}
	if out["843728390653970339"].DisplayName != "Chị Lan" || out["843728390653970339"].Avatar != "https://cdn/lan.jpg" {
		t.Fatalf("sai hồ sơ: %#v", out["843728390653970339"])
	}
	if out["5845665227953141513"].DisplayName != "Anh Nam" {
		t.Fatalf("sai hồ sơ lồng trong mảng: %#v", out["5845665227953141513"])
	}
}

// Node thiếu uid hoặc thiếu tên là mảnh dữ liệu khác; ghi vào sẽ đè hỏng hồ sơ
// đang đúng.
func TestPartialNodesAreIgnored(t *testing.T) {
	out := map[string]Profile{}
	collectProfiles(map[string]any{
		"a": map[string]any{"userId": "111"},
		"b": map[string]any{"zaloName": "không có uid"},
	}, out, 0)
	if len(out) != 0 {
		t.Fatalf("không được nhận node thiếu dữ liệu: %#v", out)
	}
}
