# Quy ước viết giáo trình

Chốt ngày 2026-09-06. Đọc file này trước khi viết bất kỳ bài nào.

## Người đọc

Thợ mới, chưa từng sửa vợt. Không giả định người đọc biết thuật ngữ, biết cầm dụng cụ,
hay đoán được bước tiếp theo. Viết sao cho người đọc làm theo được mà không cần hỏi lại.

## Nguồn nội dung

Nội dung chuyên môn đến từ hai nguồn:

1. Kendy chép lại từ lớp học nghề.
2. Tài liệu, ảnh, ghi chú Kendy đưa vào `_nguon/`.

**Claude không tự bịa kỹ thuật.** Chỗ chưa có nội dung để nguyên dấu `⬜ CHƯA CÓ — chờ Kendy`.
Sai một con số thời gian chờ keo là hỏng cây vợt của khách.

Nếu cần Claude viết nháp để Kendy sửa, Kendy phải nói rõ ở từng bài. Bản nháp đó đánh dấu
`🟨 NHÁP — Kendy chưa duyệt` cho tới khi được duyệt.

## Cách viết

- Câu ngắn, một việc một câu. Tránh câu ghép nhiều mệnh đề.
- Bước làm phải đánh số, mỗi bước là một hành động làm được ngay.
- Con số phải cụ thể: nhiệt độ, thời gian chờ, lượng keo, lực siết. Không viết "chờ một lúc".
- Thuật ngữ tiếng Anh phổ biến trong nghề thì giữ nguyên, chú thích tiếng Việt ở lần
  xuất hiện đầu tiên trong bài. Ví dụ: *delamination* (tách lớp).
- Cảnh báo nguy hiểm đặt **trước** bước làm, không đặt sau.
- Không viết văn. Không mở bài, không tổng kết lại thứ vừa nói.

## Khuôn bài Phần 3 — Các ca sửa

Mọi bài trong `03-ca-sua/` dùng 8 mục này, đúng thứ tự:

1. **Dấu hiệu nhận biết** — nhìn thấy gì, sờ thấy gì, gõ nghe thế nào
2. **Nguyên nhân** — vì sao vợt hỏng kiểu này
3. **Chọn mức sửa** — bảng tình trạng → mức, kèm dấu hiệu mức đó sẽ hỏng
4. **Vật tư và dụng cụ** — liệt kê đủ, ghi rõ loại và quy cách
5. **Các bước làm** — đánh số, ghi thời gian chờ ở đúng bước
6. **Lỗi thường gặp** — bảng: lỗi · hậu quả · cách tránh
7. **Nghiệm thu** — tiêu chí đạt. Bài nào cũng phải có bước kiểm tra cân vợt.
8. **Giá và thời gian** — công tham khảo, thời gian hoàn thành

**Bài có nhiều mức sửa** (3.2, 3.4, 3.7) tách mỗi mức thành một mục riêng giữa mục 4 và
mục lỗi, và thêm mục **"Ca vẫn KHÔNG nhận"**. Thứ tự và tên các mục còn lại giữ nguyên.

**Ba điều bắt buộc có trong mọi bài ca nặng**, đặt trước phần các bước làm: mất bảo hành
hãng · không dùng được thi đấu · vợt không trở lại như mới.

## Ảnh

- Để trong `anh/`, đặt tên theo mã bài: `3-1-nut-vien-01.jpg`, `3-1-nut-vien-02.jpg`
- Chèn vào bài kèm chú thích nói rõ ảnh cho thấy điều gì:
  `![Vết nứt chân viền, nhìn từ mặt sau](../anh/3-1-nut-vien-01.jpg)`
- Ảnh trước/sau của cùng một ca sửa đặt cạnh nhau, ghi rõ đâu là trước.

## Trạng thái bài

Đầu mỗi file có một dòng trạng thái. Sửa cả ở đó lẫn ở bảng mục lục trong `README.md`.

- `⬜ CHƯA CÓ` — mới có khung
- `🟨 NHÁP` — có nội dung, Kendy chưa duyệt
- `✅ XONG` — Kendy đã duyệt, dùng dạy được

## Quy trình làm việc

1. Kendy đưa nội dung một bài (chép tay, ảnh chụp, hoặc kể miệng).
2. Claude biên tập vào đúng khuôn, giữ nguyên mọi con số Kendy đưa.
3. Chỗ nào Kendy chưa nói tới thì để `⬜`, hỏi lại đúng chỗ đó — không tự điền.
4. Kendy đọc, sửa, đổi trạng thái sang `✅`.

Làm từng bài một. Không dựng nội dung hàng loạt.

## Xem them

- [readme.md](readme.md)
- [nguon.md](nguon.md)
- [video.md](video.md)
