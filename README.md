# zalo-kit

Private Go toolkit dùng chung cho ZChốt:

- Zalo client wrapper và mutex safety.
- QR login server-side.
- Socket lifecycle, reconnect và backoff.
- Auth/network/proxy health classification.
- Inbound event normalization.
- Send policy và account safety primitives.

Bề mặt hành động (đủ để dựng một ứng dụng chat, không chỉ gửi tin bán hàng):

| Nhóm | Tệp | Gồm |
|---|---|---|
| Gửi tin | `client/send_more.go`, `client/sendimage.go` | chữ, ảnh, tệp, video, thoại, nhãn dán, link, nhiều ảnh, trả lời trích dẫn, thu hồi, đánh dấu đã nhận/đã xem |
| Tệp trên máy | `client/media.go` | tải lên rồi gửi ảnh/tệp/ảnh động từ đĩa, danh thiếp, hạn tự xoá tin |
| Quan hệ | `client/friends.go`, `client/relations.go` | tra số điện thoại, mời/đồng ý/huỷ kết bạn, chặn, tên gợi nhớ, danh bạ, lời mời đến và đã gửi |
| Nhóm | `client/groups.go`, `client/groups_content.go` | lập/giải tán, thành viên, quản trị, đường mời, duyệt người chờ, ghim, cảm xúc, xoá tin, bình chọn, ghi chú, bảng tin, cài đặt |
| Tài khoản | `client/account.go`, `client/misc.go` | hồ sơ nick, ảnh đại diện, nhãn dán, thẻ ngân hàng, báo xấu, mã QR, trạng thái phiên, báo hiệu cuộc gọi |
| Sự kiện | `client/client.go` | `ListenWithEvents`: tin nhắn, đã nhận, đã xem, biến động nhóm, lỗi đường truyền, tải tệp xong |

⚠️ Hai giới hạn có thật của Zalo, đừng hứa với người dùng nhiều hơn:

- **Không có API đọc lịch sử cũ.** `LastMessages()` chỉ trả tin CUỐI của từng
  hội thoại. Lịch sử đầy đủ chỉ lớn dần từ lúc bắt đầu lắng nghe.
- **Các hàm đọc nội dung nhóm trả `map[string]any`** chứ không phải struct,
  vì chưa soi được dạng thật trên một phiên sống. Dựng struct từ phỏng đoán
  thì trường đoán sai bị bỏ âm thầm.

Module: `github.com/thteam47/zalo-kit`.

Thư viện này không chứa bot workflow, tenant data, prompt, memory hoặc nghiệp vụ bán hàng.
