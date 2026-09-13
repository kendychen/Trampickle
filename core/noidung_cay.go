package core

// Cây khoá nội dung: mọi câu chữ trên trang khách, xếp theo trang rồi theo
// khối, đúng thứ tự người ta thấy khi cuộn từ trên xuống. Trang quản trị
// dựng thẳng từ cây này nên thêm một khoá là xong cả ba việc: template gọi
// được, admin hiện ô nhập, và file yaml nhận được giá trị mới.
//
// Đặt tên khoá: <trang>.<khối>.<chỗ>. Đổi tên một khoá đã phát hành là làm
// mất câu chữ Kendy đã sửa (khoá cũ trong yaml thành khoá lạ, bị bỏ qua) —
// nên đặt xong thì để yên.
//
// Chữ mặc định ở đây là chữ ĐANG chạy trên trang. Sửa ở đây là đổi mặc định
// cho mọi bản cài; sửa ở /qt/noi-dung là đổi riêng cho bản của trạm.

var CayND = []TrangND{
	{
		Ma:   "chung",
		Ten:  "Dùng chung",
		MoTa: "Đầu trang, chân trang và form gửi yêu cầu — ba chỗ này hiện ở mọi trang, sửa một lần là đổi hết.",
		Nhom: []NhomND{
			{
				Ten: "Thông báo trạm chưa mở cửa hàng — để trống là tắt hẳn",
				Muc: []MucND{
					// Hiện ngay dưới địa chỉ ở /app, trang chủ và trang liên hệ —
					// đúng ba chỗ khách đọc địa chỉ rồi định phóng xe tới. Nói ở
					// chỗ khác (chân trang, trang giới thiệu) thì người đã lên
					// đường không đọc, mà người không định tới lại phải đọc.
					//
					// Địa chỉ vẫn để nguyên: nó là chỗ NHẬN vợt có thật, chỉ chưa
					// phải cửa hàng bước vào xem đồ. Giấu địa chỉ đi thì khách gửi
					// hàng biết ghi vào đâu.
					//
					// Mở cửa hàng xong thì xoá trắng ô này ở /qt/noi-dung là dải
					// biến mất khỏi cả ba trang, không phải sửa code.
					{Khoa: "chung.chua-mo", Nhan: "Câu báo — để trống thì không hiện ở đâu cả. Kẹp **hai sao** để in đậm", Dai: true,
						Mac: "**Cửa hàng đang hoàn thiện** — trạm vẫn nhận sửa vợt và giày bình thường. Muốn mang tới tận nơi thì gọi hẹn trước cho chắc."},
				},
			},
			{
				Ten: "Thanh điều hướng",
				Muc: []MucND{
					{Khoa: "chung.nav.dich-vu", Nhan: "Mục Dịch vụ", Mac: "Dịch vụ"},
					{Khoa: "chung.nav.cua-hang", Nhan: "Nav — đặt sửa online", Mac: "Đặt sửa online"},
					{Khoa: "chung.nav.quy-trinh", Nhan: "Mục Quy trình", Mac: "Quy trình"},
					{Khoa: "chung.nav.bai-viet", Nhan: "Mục Bài viết", Mac: "Bài viết"},
					{Khoa: "chung.nav.cau-hoi", Nhan: "Mục Câu hỏi", Mac: "Hỏi đáp"},
					{Khoa: "chung.nav.gioi-thieu", Nhan: "Mục Giới thiệu", Mac: "Giới thiệu"},
					{Khoa: "chung.nav.ve-chung-toi", Nhan: "Mục Về chúng tôi", Mac: "Về chúng tôi"},
					{Khoa: "chung.nav.lien-he", Nhan: "Mục Liên hệ", Mac: "Liên hệ"},
					{Khoa: "chung.nav.tra-cuu", Nhan: "Nút tra cứu", Mac: "Tra cứu đơn"},
				},
			},
			{
				Ten: "Mức cân trạm cam kết",
				Muc: []MucND{
					{Khoa: "chung.can.muc", Nhan: "Mức tăng cân nói cho khách (hiện ở trang chủ, dịch vụ, quy trình, giới thiệu)",
						Mac: "10–15 g tùy vợt"},
				},
			},
			{
				Ten: "Nút bấm dùng ở nhiều trang",
				Muc: []MucND{
					{Khoa: "chung.nut.gui-anh", Nhan: "Nút gửi ảnh", Mac: "Gửi ảnh chỗ hỏng"},
					{Khoa: "chung.nut.quy-trinh", Nhan: "Nút xem quy trình", Mac: "Xem quy trình"},
					{Khoa: "chung.nut.dich-vu", Nhan: "Nút danh sách dịch vụ", Mac: "Danh sách dịch vụ"},
					{Khoa: "chung.nut.tra-cuu", Nhan: "Nút tra cứu đơn", Mac: "Tra cứu đơn"},
					{Khoa: "chung.nut.lien-he", Nhan: "Nút liên hệ trạm", Mac: "Liên hệ trạm"},
				},
			},
			{
				Ten: "Ô dịch vụ (lưới việc ở trang chủ và trang dịch vụ)",
				Muc: []MucND{
					{Khoa: "chung.the.bao-hanh", Nhan: "Nhãn bảo hành — {thang} là số tháng khai ở /qt/dich-vu", Mac: "BH {thang} tháng"},
				},
			},
			{
				Ten: "Chân trang",
				Muc: []MucND{
					{Khoa: "chung.chan.gioi-thieu", Nhan: "Câu giới thiệu (đứng sau dòng mô tả thương hiệu)", Dai: true,
						Mac: "Khám vợt xong báo giá ngay — anh/chị chốt rồi thợ mới làm."},
					{Khoa: "chung.chan.cot-1", Nhan: "Tên cột 1", Mac: "Dịch vụ"},
					{Khoa: "chung.chan.dich-vu-ds", Nhan: "Cột 1 — dòng 1", Mac: "Danh sách dịch vụ"},
					{Khoa: "chung.chan.quy-trinh", Nhan: "Cột 1 — dòng 2", Mac: "Quy trình"},
					{Khoa: "chung.chan.bai-viet", Nhan: "Cột 1 — dòng 3", Mac: "Bài viết"},
					{Khoa: "chung.chan.tra-cuu", Nhan: "Cột 1 — dòng 4", Mac: "Tra cứu đơn sửa"},
					{Khoa: "chung.chan.cau-hoi", Nhan: "Cột 1 — dòng 5", Mac: "Câu hỏi thường gặp"},
					{Khoa: "chung.chan.cot-2", Nhan: "Tên cột 2", Mac: "Liên hệ"},
					{Khoa: "chung.chan.chua-co", Nhan: "Khi chưa điền thông tin liên hệ", Mac: "Đang cập nhật"},
					{Khoa: "chung.chan.cot-3", Nhan: "Tên cột 3", Mac: "Khác"},
					{Khoa: "chung.chan.chinh-sach", Nhan: "Cột 3 — dòng 1", Mac: "Chính sách bảo mật"},
					{Khoa: "chung.chan.day", Nhan: "Dòng cuối cùng (đứng sau © và tên trạm)", Dai: true,
						Mac: "Nhận sửa vợt pickleball và thay đế giày thể thao."},
				},
			},
			{
				Ten: "Form gửi yêu cầu",
				Muc: []MucND{
					{Khoa: "form.ten", Nhan: "Nhãn ô tên", Mac: "Tên"},
					{Khoa: "form.lien-he", Nhan: "Nhãn ô liên hệ", Mac: "Điện thoại hoặc Zalo *"},
					{Khoa: "form.email", Nhan: "Nhãn ô email", Mac: "Email"},
					{Khoa: "form.email-hd", Nhan: "Gợi ý dưới ô email", Dai: true,
						Mac: "Không bắt buộc. Có email thì em gửi mã yêu cầu và lịch hẹn khám về đó, khỏi lo quên."},
					{Khoa: "form.email-goi-y", Nhan: "Chữ mờ trong ô email", Mac: "ten@gmail.com"},
					{Khoa: "form.hang", Nhan: "Nhãn ô hãng vợt/giày", Mac: "Vợt hoặc giày — hãng gì, đời nào"},
					{Khoa: "form.hang-goi-y", Nhan: "Chữ mờ trong ô hãng", Mac: "Ví dụ: Joola Perseus 16mm, hoặc Nike Pegasus 40"},
					{Khoa: "form.mo-ta", Nhan: "Nhãn ô mô tả hỏng", Mac: "Đang bị gì *"},
					{Khoa: "form.mo-ta-hd", Nhan: "Gợi ý dưới ô mô tả", Dai: true,
						Mac: "Vợt: nứt ở đâu, có tiếng lạ không, còn ăn bóng không. Giày: bong đế chỗ nào, mòn tới đâu."},
					{Khoa: "form.tep", Nhan: "Nhãn ô tải ảnh/video", Mac: "Ảnh hoặc video chỗ hỏng"},
					{Khoa: "form.tep-hd", Nhan: "Gợi ý dưới ô tải tệp", Dai: true,
						Mac: "Tối đa 5 tệp. Ảnh dưới 12MB, video dưới 40MB (khoảng 10–20 giây)."},
					{Khoa: "form.nut", Nhan: "Nút gửi", Mac: "Gửi cho trạm"},
					{Khoa: "form.nut-nang", Nhan: "Nút gửi khi có tệp quá nặng", Mac: "Có tệp quá nặng"},
					{Khoa: "form.qua-video", Nhan: "Báo video quá nặng", Mac: "⚠ quá 40MB, quay ngắn lại giúp em"},
					{Khoa: "form.qua-anh", Nhan: "Báo ảnh quá nặng", Mac: "⚠ quá 12MB"},
					{Khoa: "form.qua-so-tep", Nhan: "Báo gửi quá 5 tệp", Mac: "⚠ Chỉ nhận 5 tệp đầu tiên."},
				},
			},
		},
	},
	{
		Ma:   "trangchu",
		Ten:  "Trang chủ",
		MoTa: "Xếp đúng thứ tự khách cuộn từ trên xuống.",
		Nhom: []NhomND{
			{
				Ten: "Tấm đầu trang",
				Muc: []MucND{
					{Khoa: "trangchu.hero.nhan", Nhan: "Dòng nhãn nhỏ trên cùng", Mac: "Sửa chữa vợt Pickleball · Thay đế giày thể thao"},
					{Khoa: "trangchu.hero.h1", Nhan: "Dòng chữ to nhất trang",
						Mac:   "Gửi ảnh xem trước. Khám xong là có giá ngay.",
						MacOn: "Gửi vợt tới trạm. Sửa xong trạm gửi về tận nhà."},
					{Khoa: "trangchu.hero.dan", Nhan: "Đoạn dẫn dưới dòng chữ to", Dai: true,
						Mac:   "Gửi ảnh, video và kể chỗ hỏng — em xem rồi nhận xét trong ngày: cây này thuộc việc gì, làm được hay không, mất mấy ngày. Chưa có con số ở bước này. Thấy ổn thì gửi vợt tới; em khám xong là báo giá ngay tại chỗ, và giá đã chốt là giá lúc nhận vợt về, không gọi lại giữa chừng.",
						MacOn: "Đặt đơn trên web, lấy mã rồi gửi vợt tới trạm. Vợt tới nơi em khám và báo giá, kèm ảnh chụp chỗ hỏng — anh/chị chốt rồi thợ mới làm. Giá đã chốt là giá cuối, không gọi lại giữa chừng."},
					{Khoa: "trangchu.hero.dia-chi-nhan", Nhan: "Nhãn trước địa chỉ ở dòng liên hệ", Mac: "Địa chỉ", MacOn: "Gửi vợt tới"},
					{Khoa: "trangchu.hero.nut1", Nhan: "Nút chính", Mac: "Gửi ảnh để em xem trước"},
					{Khoa: "trangchu.hero.nut2", Nhan: "Nút phụ", Mac: "Xem bảng việc và thời gian"},
					{Khoa: "trangchu.herodv.nhan", Nhan: "Cột phải hero — tiêu đề danh sách việc", Mac: "Trạm nhận những việc này"},
					{Khoa: "trangchu.herodv.tatca", Nhan: "Cột phải hero — dòng cuối", Mac: "Xem tất cả và thời gian làm"},
				},
			},
			{
				Ten: "Dải ba con số",
				Muc: []MucND{
					{Khoa: "trangchu.so.1-nhan", Nhan: "Ô 1 — nhãn", Mac: "Đang nhận"},
					{Khoa: "trangchu.so.1-chu", Nhan: "Ô 1 — chú thích (con số là số dịch vụ đang mở, tự đếm)", Mac: "dịch vụ — mỗi việc một quy trình viết sẵn"},
					{Khoa: "trangchu.so.2-nhan", Nhan: "Ô 2 — nhãn", Mac: "Trả lời tin nhắn"},
					{Khoa: "trangchu.so.2-so", Nhan: "Ô 2 — con số", Mac: "24"},
					{Khoa: "trangchu.so.2-dv", Nhan: "Ô 2 — đơn vị (chữ nhỏ sau số)", Mac: "h"},
					{Khoa: "trangchu.so.2-chu", Nhan: "Ô 2 — chú thích", Mac: "tính từ lúc anh/chị gửi ảnh chỗ hỏng"},
					{Khoa: "trangchu.so.3-nhan", Nhan: "Ô 3 — nhãn", Mac: "Trả trước khi chốt giá"},
					{Khoa: "trangchu.so.3-so", Nhan: "Ô 3 — con số", Mac: "0"},
					{Khoa: "trangchu.so.3-dv", Nhan: "Ô 3 — đơn vị", Mac: "đ"},
					{Khoa: "trangchu.so.3-chu", Nhan: "Ô 3 — chú thích", Mac: "khám vợt, báo giá và tư vấn đều không tính tiền"},
				},
			},
			{
				Ten: "Phiếu khám ví dụ",
				Muc: []MucND{
					{Khoa: "trangchu.phieu.dinh", Nhan: "Tên phiếu", Mac: "Phiếu khám"},
					{Khoa: "trangchu.phieu.dinh-phu", Nhan: "Chữ nhỏ cạnh tên phiếu", Mac: "ví dụ"},
					{Khoa: "trangchu.phieu.d1", Nhan: "Dòng 1 — hỏng gì", Mac: "Viền hở keo, dài 6 cm"},
					{Khoa: "trangchu.phieu.d1-ket", Nhan: "Dòng 1 — kết luận", Mac: "dán lại"},
					{Khoa: "trangchu.phieu.d2", Nhan: "Dòng 2 — hỏng gì", Mac: "Thủng mặt, 1 điểm"},
					{Khoa: "trangchu.phieu.d2-ket", Nhan: "Dòng 2 — kết luận", Mac: "vá được"},
					{Khoa: "trangchu.phieu.d3", Nhan: "Dòng 3 — hỏng gì", Mac: "Lõi dưới chỗ thủng"},
					{Khoa: "trangchu.phieu.d3-ket", Nhan: "Dòng 3 — kết luận", Mac: "còn nguyên"},
					{Khoa: "trangchu.phieu.chot", Nhan: "Dòng cuối phiếu", Mac: "Khám xong báo giá — chốt rồi mới làm"},
					{Khoa: "trangchu.phieu.chot-ket", Nhan: "Dòng cuối — vế phải", Mac: "2 ngày"},
					{Khoa: "trangchu.phieu.ghi", Nhan: "Câu giải thích dưới phiếu", Dai: true,
						Mac: "Khám xong anh/chị cầm được tờ này: hỏng những gì, việc nào làm được, mất mấy ngày. Con số đi kèm ngay lúc đó, không phải đợi."},
				},
			},
			{
				Ten: "Mục báo giá",
				Muc: []MucND{
					{Khoa: "trangchu.baogia.nhan", Nhan: "Nhãn nhỏ", Mac: "Báo giá"},
					{Khoa: "trangchu.baogia.h2", Nhan: "Tiêu đề mục",
						Mac:   "Giá ra tại chỗ, ngay sau khi khám",
						MacOn: "Vợt tới nơi, khám xong là có giá"},
					{Khoa: "trangchu.baogia.dan", Nhan: "Đoạn dẫn", Dai: true,
						Mac:   "Thứ làm người ta ngại mang vợt đi sửa không phải là tiền, mà là không biết bao giờ mới biết hết bao nhiêu. Nên ở đây khám xong là ra số ngay, và con số ấy đi trước công việc chứ không đi sau.",
						MacOn: "Thứ làm người ta ngại gửi vợt đi sửa không phải là tiền, mà là gửi đi rồi không biết bao giờ mới biết hết bao nhiêu. Nên vợt tới nơi là khám và ra số ngay, con số ấy đi trước công việc chứ không đi sau."},
					{Khoa: "trangchu.baogia.y1", Nhan: "Gạch đầu dòng 1", Dai: true,
						Mac: "**Khám trước, ra giá sau.** Gõ mặt, soi mép, kiểm tra cán — rồi mới nói tiền."},
					{Khoa: "trangchu.baogia.y2", Nhan: "Gạch đầu dòng 2", Dai: true,
						Mac: "**Khám không tính phí,** kể cả khi anh/chị nghe giá xong quyết định không sửa."},
					{Khoa: "trangchu.baogia.y3", Nhan: "Gạch đầu dòng 3", Dai: true,
						Mac: "**Chốt rồi mới làm.** Chưa đồng ý thì vợt vẫn nguyên, không mất đồng nào."},
				},
			},
			{
				Ten: "Mục vì sao phải khám",
				Muc: []MucND{
					{Khoa: "trangchu.kham.chu-thich", Nhan: "Chú thích dưới bản vẽ mặt vợt", Dai: true,
						Mac: "Mặt vợt là hai lớp carbon ép lên một lõi tổ ong polymer. Hư hại nằm ở lớp ngoài hay đã ăn vào lõi là hai việc khác nhau, và giá cũng khác nhau — nên phải khám rồi mới ra được con số."},
					{Khoa: "trangchu.kham.nhan", Nhan: "Nhãn nhỏ", Mac: "Vì sao phải khám"},
					{Khoa: "trangchu.kham.h2", Nhan: "Tiêu đề mục", Mac: "Cây vợt nhìn ngoài lành lặn vẫn có thể đã hỏng bên trong"},
					{Khoa: "trangchu.kham.dan", Nhan: "Đoạn dẫn", Dai: true,
						Mac: "Hai kiểu hư hại nặng nhất — sập lõi và tách lớp — đều nằm dưới lớp mặt còn nguyên vẹn. Ảnh không cho thấy, mắt thường cũng không. Chỉ khám tận tay mới ra, và đó là lý do con số chỉ xuất hiện sau bước khám."},
					{Khoa: "trangchu.kham.y1", Nhan: "Gạch đầu dòng 1", Dai: true,
						Mac: "**Gõ khắp mặt vợt.** Vùng lành kêu đanh và đều. Vùng kêu bộp là tách lớp hoặc sập lõi bên dưới."},
					{Khoa: "trangchu.kham.y2", Nhan: "Gạch đầu dòng 2", Dai: true,
						Mac: "**Soi ngược sáng dọc mép.** Khe hở giữa lớp mặt và lõi hiện ra thành một vệt tối mảnh."},
					{Khoa: "trangchu.kham.y3", Nhan: "Gạch đầu dòng 3", Dai: true,
						Mac: "**Chụp lại chỗ nghi ngờ.** Ảnh này đi kèm đơn, để anh/chị thấy đúng thứ em thấy."},
					{Khoa: "trangchu.kham.nut", Nhan: "Nút dẫn sang bài viết", Mac: "Đọc kỹ hơn: khám vợt trước khi sửa"},
				},
			},
			{
				Ten: "Mục nhận sửa",
				Muc: []MucND{
					{Khoa: "trangchu.dv.nhan", Nhan: "Nhãn nhỏ", Mac: "Nhận sửa"},
					{Khoa: "trangchu.dv.h2", Nhan: "Tiêu đề mục", Mac: "Làm ít việc, nhưng việc nào cũng có quy trình"},
					{Khoa: "trangchu.dv.dan", Nhan: "Đoạn dẫn", Dai: true,
						Mac: "Bấm vào từng mục để xem chi tiết: làm gì, mất bao lâu, bảo hành bao lâu."},
					{Khoa: "trangchu.dv.noi-bat", Nhan: "Nhãn việc nổi bật", Mac: "Nhiều khách chọn"},
					{Khoa: "trangchu.dv.khac-h3", Nhan: "Ô cuối lưới — tiêu đề", Mac: "Không thấy việc của mình?"},
					{Khoa: "trangchu.dv.khac-chip", Nhan: "Ô cuối lưới — nhãn tròn", Mac: "Hỏi trước"},
					{Khoa: "trangchu.dv.khac-chu", Nhan: "Ô cuối lưới — mô tả", Dai: true,
						Mac: "Còn nhiều việc chưa dựng thành mục riêng. Gửi ảnh, em xem rồi hẹn khám."},
					{Khoa: "trangchu.dv.khac-chan", Nhan: "Ô cuối lưới — dòng chân", Mac: "Trả lời trong ngày"},
				},
			},
			{
				Ten: "Mục giày bong đế",
				Muc: []MucND{
					{Khoa: "trangchu.giay.chu-thich", Nhan: "Chú thích dưới bản vẽ đế giày", Dai: true,
						Mac: "Đế ngoài bong khỏi đế giữa là hư hại thay được — phần thân giày và lớp đệm bên trên không bị đụng tới."},
					{Khoa: "trangchu.giay.nhan", Nhan: "Nhãn nhỏ", Mac: "Không chỉ vợt"},
					{Khoa: "trangchu.giay.h2", Nhan: "Tiêu đề mục", Mac: "Giày bong đế cũng nhận"},
					{Khoa: "trangchu.giay.dan", Nhan: "Đoạn dẫn", Dai: true,
						Mac: "Giày pickleball hỏng sớm nhất ở đế, vì môn này trượt ngang nhiều hơn chạy thẳng. Đế ngoài bong hoặc mòn hết gai mà thân giày còn tốt thì thay đế đáng tiền hơn mua đôi mới."},
					{Khoa: "trangchu.giay.y1", Nhan: "Gạch đầu dòng 1", Dai: true,
						Mac: "**Thay được:** đế ngoài bong khỏi đế giữa, mòn trơ, nứt ngang phần mũi."},
					{Khoa: "trangchu.giay.y2", Nhan: "Gạch đầu dòng 2", Dai: true,
						Mac: "**Xem thêm:** đế giữa đã xẹp thì em nói rõ thay đế lấy lại được gì và không lấy lại được gì, để anh/chị tự cân nhắc."},
					{Khoa: "trangchu.giay.y3", Nhan: "Gạch đầu dòng 3", Dai: true,
						Mac: "Gửi ảnh chụp đế nhìn từ dưới lên và ảnh nhìn ngang, em xem là hẹn được lịch khám."},
					{Khoa: "trangchu.giay.nut", Nhan: "Nút", Mac: "Gửi ảnh đôi giày"},
				},
			},
			{
				Ten: "Mục bốn bước",
				Muc: []MucND{
					{Khoa: "trangchu.qt.nhan", Nhan: "Nhãn nhỏ", Mac: "Quy trình"},
					{Khoa: "trangchu.qt.h2", Nhan: "Tiêu đề mục", Mac: "Bốn bước, không có bước nào giấu"},
					{Khoa: "trangchu.qt.b1-so", Nhan: "Bước 1 — số", Mac: "BƯỚC 1"},
					{Khoa: "trangchu.qt.b1-ten", Nhan: "Bước 1 — tên", Mac: "Gửi ảnh, nghe nhận xét"},
					{Khoa: "trangchu.qt.b1-chu", Nhan: "Bước 1 — mô tả", Dai: true,
						Mac: "Chụp chỗ hỏng, quay thêm nếu có tiếng lạ, nhắn qua form hoặc Zalo. Em xem rồi trả lời trong ngày: cây này thuộc việc gì, làm được không, mất mấy ngày. Chưa có giá ở bước này."},
					{Khoa: "trangchu.qt.b2-so", Nhan: "Bước 2 — số", Mac: "BƯỚC 2"},
					{Khoa: "trangchu.qt.b2-ten", Nhan: "Bước 2 — tên", Mac: "Gửi vợt và khám"},
					{Khoa: "trangchu.qt.b2-chu", Nhan: "Bước 2 — mô tả", Dai: true,
						Mac:   "Thấy ổn thì anh/chị gửi tới hoặc mang tới. Em gõ mặt, soi mép, kiểm tra cán — khám xong là có giá ngay.",
						MacOn: "Thấy ổn thì anh/chị đặt đơn, ghi mã lên kiện rồi gửi tới trạm. Em gõ mặt, soi mép, kiểm tra cán — khám xong là có giá ngay."},
					{Khoa: "trangchu.qt.b3-so", Nhan: "Bước 3 — số", Mac: "BƯỚC 3"},
					{Khoa: "trangchu.qt.b3-ten", Nhan: "Bước 3 — tên", Mac: "Chốt giá rồi mới làm"},
					{Khoa: "trangchu.qt.b3-chu", Nhan: "Bước 3 — mô tả", Dai: true,
						Mac: "Anh/chị đồng ý thì em bắt tay vào; không đồng ý thì nhận vợt về nguyên trạng, không tính tiền."},
					{Khoa: "trangchu.qt.b4-so", Nhan: "Bước 4 — số", Mac: "BƯỚC 4"},
					{Khoa: "trangchu.qt.b4-ten", Nhan: "Bước 4 — tên", Mac: "Nghiệm thu và bàn giao"},
					{Khoa: "trangchu.qt.b4-chu", Nhan: "Bước 4 — mô tả", Dai: true,
						Mac: "Sửa xong em chụp lại chỗ đã làm, gửi ảnh trước khi giao. Bảo hành tính từ ngày giao."},
				},
			},
			{
				Ten: "Mục ba cam kết",
				Muc: []MucND{
					{Khoa: "trangchu.ck.nhan", Nhan: "Nhãn nhỏ", Mac: "Cam kết"},
					{Khoa: "trangchu.ck.h2", Nhan: "Tiêu đề mục", Mac: "Ba điều em chịu trách nhiệm"},
					{Khoa: "trangchu.ck.1-ten", Nhan: "Cam kết 1 — tên", Mac: "Giá chốt trước khi làm là giá cuối"},
					{Khoa: "trangchu.ck.1-chu", Nhan: "Cam kết 1 — mô tả", Dai: true,
						Mac: "Không có khoản phát sinh gọi thêm giữa chừng. Nếu làm tới nơi mới lộ ra việc khác, em báo lại và chờ anh/chị quyết, chưa quyết thì chưa làm."},
					{Khoa: "trangchu.ck.2-ten", Nhan: "Cam kết 2 — tên", Mac: "Chưa đồng ý giá thì không mất đồng nào"},
					{Khoa: "trangchu.ck.2-chu", Nhan: "Cam kết 2 — mô tả", Dai: true,
						Mac: "Khám vợt, báo giá và tư vấn đều miễn phí. Anh/chị nghe giá xong đổi ý thì nhận vợt về nguyên trạng."},
					{Khoa: "trangchu.ck.3-ten", Nhan: "Cam kết 3 — tên", Mac: "Bảo hành ghi trên đơn, không nói miệng"},
					{Khoa: "trangchu.ck.3-chu", Nhan: "Cam kết 3 — mô tả", Dai: true,
						Mac: "Mỗi đơn có mã tra cứu. Ngày hết bảo hành in trong đơn, anh/chị tra bất cứ lúc nào."},
				},
			},
			{
				Ten: "Mục gửi yêu cầu",
				Muc: []MucND{
					{Khoa: "trangchu.gui.ok", Nhan: "Câu báo đã nhận được yêu cầu", Dai: true,
						Mac: "**Em nhận được rồi ạ.** Em xem ảnh rồi nhắn lại trong ngày: cây này thuộc việc gì, làm được hay không, mất mấy ngày. Thấy ổn thì lúc đó gửi vợt."},
					{Khoa: "trangchu.gui.ma-nhan", Nhan: "Nhãn trên ô mã yêu cầu", Mac: "Mã yêu cầu của anh/chị"},
					{Khoa: "trangchu.gui.mail-co", Nhan: "Câu báo đã gửi mã qua email — chỗ {email} là địa chỉ khách điền", Dai: true,
						Mac: "Em gửi mã này về **{email}** luôn rồi ạ — không thấy thư thì anh/chị ngó hộp Spam giúp em."},
					{Khoa: "trangchu.gui.mail-khong", Nhan: "Câu thay thế khi khách không để email", Dai: true,
						Mac: "Chụp màn hình hoặc lưu lại mã này."},
					{Khoa: "trangchu.gui.tra-cuu", Nhan: "Câu dặn cách tra cứu", Dai: true,
						Mac: "Gõ mã cùng 4 số cuối điện thoại ở [trang tra cứu](/tra-cuu) là xem được yêu cầu đang tới đâu — và sau khi em dựng thành đơn thì vẫn tra bằng đúng mã ấy."},
					{Khoa: "trangchu.gui.nhan", Nhan: "Nhãn nhỏ", Mac: "Gửi yêu cầu"},
					{Khoa: "trangchu.gui.h2", Nhan: "Tiêu đề mục", Mac: "Gửi ảnh chỗ hỏng, em xem rồi nhận xét"},
					{Khoa: "trangchu.gui.dan", Nhan: "Đoạn dẫn trên form", Dai: true,
						Mac: "Vợt nứt viền, bong lớp — hay giày bong đế. Chụp rõ chỗ hỏng, quay thêm 10–15 giây nếu có tiếng lạ khi đánh. Em xem rồi trả lời trong ngày: cây vợt này thuộc việc gì, làm được hay không, mất mấy ngày. Chưa có con số ở bước này — số chỉ có sau khi khám tận tay. Thấy ổn thì gửi vợt tới."},
					{Khoa: "trangchu.canh.1-h3", Nhan: "Thẻ bên phải 1 — tiêu đề", Mac: "Đã gửi vợt rồi?"},
					{Khoa: "trangchu.canh.1-chu", Nhan: "Thẻ bên phải 1 — mô tả", Dai: true,
						Mac: "Tra bằng mã đơn trên phiếu và 4 số cuối điện thoại — thấy vợt đang ở bước nào, hẹn trả ngày nào, còn thiếu bao nhiêu tiền."},
					{Khoa: "trangchu.canh.1-nut", Nhan: "Thẻ bên phải 1 — nút", Mac: "Tra cứu đơn"},
					{Khoa: "trangchu.canh.2-h3", Nhan: "Thẻ bên phải 2 — tiêu đề", Mac: "Muốn biết giá?"},
					{Khoa: "trangchu.canh.2-chu", Nhan: "Thẻ bên phải 2 — mô tả", Dai: true,
						Mac:   "Mang vợt tới trạm là khám ngay trước mặt và có giá trong buổi. Nhắn trước một câu để em xếp lịch, khỏi phải ngồi chờ.",
						MacOn: "Bảng việc có giá khởi điểm của từng món. Số chốt thì phải khám tận tay mới có — vợt tới nơi là em báo ngay trong ngày."},
					{Khoa: "trangchu.canh.2-nut", Nhan: "Thẻ bên phải 2 — nút", Mac: "Danh sách dịch vụ"},
				},
			},
		},
	},
	{
		Ma:   "dichvu",
		Ten:  "Trang dịch vụ",
		MoTa: "Trang liệt kê các việc trạm nhận. Tên và mô tả từng dịch vụ nằm ở mục Dịch vụ, không sửa ở đây.",
		Nhom: []NhomND{
			{
				Ten: "Đầu trang",
				Muc: []MucND{
					{Khoa: "dichvu.nhan", Nhan: "Nhãn nhỏ", Mac: "Dịch vụ"},
					{Khoa: "dichvu.h1", Nhan: "Tiêu đề lớn — {so} là số dịch vụ đang mở", Mac: "Trạm đang nhận {so} việc"},
					{Khoa: "dichvu.dan", Nhan: "Đoạn dẫn", Dai: true,
						Mac: "Danh sách này ngắn có chủ ý. Làm ít việc nhưng mỗi việc có quy trình riêng thì kết quả đều tay hơn là nhận tất rồi làm theo cảm tính."},
				},
			},
			{
				Ten: "Ô cuối lưới — không thấy việc của mình",
				Muc: []MucND{
					{Khoa: "dichvu.hoi.h3", Nhan: "Tiêu đề ô", Mac: "Không thấy việc của mình?"},
					{Khoa: "dichvu.hoi.chip", Nhan: "Nhãn góc ô", Mac: "Hỏi trước"},
					{Khoa: "dichvu.hoi.chu", Nhan: "Mô tả trong ô", Dai: true,
						Mac: "Có ca em nhận nhưng chưa dựng thành mục riêng. Gửi ảnh, em xem rồi nói thẳng."},
					{Khoa: "dichvu.hoi.chan", Nhan: "Dòng chân ô", Mac: "Trả lời trong ngày"},
				},
			},
			{
				Ten: "Khối việc trạm không nhận",
				Muc: []MucND{
					{Khoa: "dichvu.tuchoi.nhan", Nhan: "Nhãn nhỏ", Mac: "Không nhận"},
					{Khoa: "dichvu.tuchoi.h2", Nhan: "Tiêu đề", Mac: "Có ca em nói thẳng là không sửa được"},
					{Khoa: "dichvu.tuchoi.chu", Nhan: "Đoạn dẫn", Dai: true,
						Mac: "Nói trước cho đỡ mất công đóng gói gửi đi. Tên việc và lý do ở dưới lấy từ bảng giá, sửa ở mục Dịch vụ."},
					{Khoa: "dichvu.tuchoi.link", Nhan: "Chữ trên link sang bài", Mac: "Vì sao, và cách tự kiểm tra ở nhà →"},
				},
			},
		},
	},
	{
		Ma:   "dichvumot",
		Ten:  "Trang một dịch vụ",
		MoTa: "Khung chung cho mọi dịch vụ. Tên việc, số ngày làm, tháng bảo hành lấy từ mục Dịch vụ.",
		Nhom: []NhomND{
			{
				Ten: "Đầu trang",
				Muc: []MucND{
					{Khoa: "dichvumot.quay-lai", Nhan: "Link quay lại", Mac: "← Tất cả dịch vụ"},
					{Khoa: "dichvumot.dieu-kien", Nhan: "Chữ đứng trước điều kiện nhận", Mac: "Điều kiện nhận:"},
					{Khoa: "dichvumot.tuchoi.chip", Nhan: "Nhãn trên trang ca không nhận", Mac: "Trạm không nhận việc này"},
					{Khoa: "dichvumot.tuchoi.viSao", Nhan: "Chữ đứng trước lý do không nhận", Mac: "Lý do:"},
				},
			},
			{
				Ten: "Dải ba con số",
				Muc: []MucND{
					{Khoa: "dichvumot.so.1-n", Nhan: "Ô 1 — nhãn", Mac: "Thời gian làm"},
					{Khoa: "dichvumot.so.1-dv", Nhan: "Ô 1 — đơn vị đi sau số", Mac: "ngày"},
					{Khoa: "dichvumot.so.1-t", Nhan: "Ô 1 — chú thích", Mac: "kể từ lúc chốt giá"},
					{Khoa: "dichvumot.so.2-n", Nhan: "Ô 2 — nhãn", Mac: "Bảo hành"},
					{Khoa: "dichvumot.so.2-dv", Nhan: "Ô 2 — đơn vị đi sau số", Mac: "tháng"},
					{Khoa: "dichvumot.so.2-t", Nhan: "Ô 2 — chú thích", Mac: "tính từ ngày giao vợt"},
					{Khoa: "dichvumot.so.3-n", Nhan: "Ô 3 — nhãn", Mac: "Tăng cân tối đa"},
					{Khoa: "dichvumot.so.3-t", Nhan: "Ô 3 — chú thích", Mac: "vượt mức này thì không lấy tiền công"},
				},
			},
			{
				Ten: "Quy trình riêng",
				Muc: []MucND{
					{Khoa: "dichvumot.qt.nhan", Nhan: "Nhãn nhỏ", Mac: "Quy trình riêng"},
					{Khoa: "dichvumot.qt.h2", Nhan: "Tiêu đề mục", Mac: "Cách trạm làm việc này"},
					{Khoa: "dichvumot.qt.b1", Nhan: "Bước 1", Dai: true,
						Mac: "Cân vợt trên cân điện tử, ghi số cân trước vào đơn."},
					{Khoa: "dichvumot.qt.b2", Nhan: "Bước 2", Dai: true,
						Mac: "Soi kỹ chỗ hỏng. Nếu phát hiện thêm hư hại nằm trong danh sách từ chối, em dừng và trả vợt."},
					{Khoa: "dichvumot.qt.b3", Nhan: "Bước 3", Dai: true,
						Mac: "Báo giá trọn gói cho đúng ca này. Anh/chị đồng ý thì em mới bắt đầu."},
					{Khoa: "dichvumot.qt.b4", Nhan: "Bước 4 — {ten} là tên dịch vụ", Dai: true,
						Mac: "Làm theo quy trình riêng của {ten}."},
					{Khoa: "dichvumot.qt.b5", Nhan: "Bước 5", Dai: true,
						Mac: "Cân lại, chụp ảnh sau, ghi chênh cân vào đơn rồi bàn giao."},
				},
			},
			{
				Ten: "Thẻ bên phải",
				Muc: []MucND{
					{Khoa: "dichvumot.gia.chu", Nhan: "Dòng nhỏ dưới con số giá", Dai: true,
						Mac: "Đây là giá sàn. Khám vợt xong trạm báo con số chính thức — anh/chị chốt rồi thợ mới làm."},
					{Khoa: "dichvumot.gui.h3", Nhan: "Tiêu đề thẻ", Mac: "Gửi vợt cho trạm"},
					{Khoa: "dichvumot.gui.chu", Nhan: "Mô tả", Dai: true,
						Mac: "Chụp chỗ hỏng gửi trước — em xem ảnh rồi trả lời có nhận hay không, trong ngày."},
					{Khoa: "dichvumot.gui.nut-2", Nhan: "Nút thứ hai", Mac: "Xem quy trình trạm"},
				},
			},
			{
				Ten: "Lưới dịch vụ khác",
				Muc: []MucND{
					{Khoa: "dichvumot.khac.nhan", Nhan: "Nhãn nhỏ", Mac: "Dịch vụ khác"},
					{Khoa: "dichvumot.tatca.h3", Nhan: "Ô xem tất cả — tiêu đề", Mac: "Xem tất cả dịch vụ"},
					{Khoa: "dichvumot.tatca.chip", Nhan: "Ô xem tất cả — nhãn góc", Mac: "Danh sách"},
				},
			},
		},
	},
	{
		Ma:   "quytrinh",
		Ten:  "Trang quy trình",
		MoTa: "Năm bước từ lúc khách nhắn tới lúc nhận vợt về, và hai đường đưa vợt tới trạm.",
		Nhom: []NhomND{
			{
				Ten: "Đầu trang",
				Muc: []MucND{
					{Khoa: "quytrinh.nhan", Nhan: "Nhãn nhỏ", Mac: "Quy trình"},
					{Khoa: "quytrinh.h1", Nhan: "Tiêu đề lớn", Mac: "Từ lúc nhắn tin đến lúc cầm vợt về"},
					{Khoa: "quytrinh.dan", Nhan: "Đoạn dẫn", Dai: true,
						Mac: "Năm bước. Giá ra ở bước 2 — khám xong vợt là có ngay con số, và anh/chị chốt trước khi em động vào vợt. Không có khoản nào xuất hiện lúc trả."},
				},
			},
			{
				Ten: "Năm bước",
				Muc: []MucND{
					{Khoa: "quytrinh.b1.n", Nhan: "Bước 1 — số", Mac: "BƯỚC 1"},
					{Khoa: "quytrinh.b1.h4", Nhan: "Bước 1 — tên", Mac: "Gửi ảnh, nghe nhận xét"},
					{Khoa: "quytrinh.b1.chu", Nhan: "Bước 1 — mô tả", Dai: true,
						Mac: "Chụp chỗ hỏng, quay thêm 10–15 giây nếu có tiếng lạ, gửi qua form hoặc Zalo. Em xem rồi trả lời trong ngày: cây vợt này thuộc việc gì, làm được hay không, mất mấy ngày. Chưa có con số ở bước này — ba thứ quyết định con số đều nằm dưới lớp viền."},
					{Khoa: "quytrinh.b2.n", Nhan: "Bước 2 — số", Mac: "BƯỚC 2"},
					{Khoa: "quytrinh.b2.h4", Nhan: "Bước 2 — tên", Mac: "Gửi vợt và khám"},
					{Khoa: "quytrinh.b2.chu", Nhan: "Bước 2 — mô tả", Dai: true,
						Mac: "Nghe nhận xét thấy ổn thì anh/chị gửi vợt, hoặc mang tới tận nơi. Em khám: gõ khắp mặt, soi ngược sáng dọc mép, kiểm tra mối nối cán. Khám xong là có giá ngay."},
					{Khoa: "quytrinh.b3.n", Nhan: "Bước 3 — số", Mac: "BƯỚC 3"},
					{Khoa: "quytrinh.b3.h4", Nhan: "Bước 3 — tên", Mac: "Chốt giá rồi mới làm"},
					{Khoa: "quytrinh.b3.chu", Nhan: "Bước 3 — mô tả", Dai: true,
						Mac: "Em báo con số và số ngày. Anh/chị đồng ý thì em bắt tay vào; không đồng ý thì nhận vợt về nguyên trạng, không tính tiền khám."},
					{Khoa: "quytrinh.b4.n", Nhan: "Bước 4 — số", Mac: "BƯỚC 4"},
					{Khoa: "quytrinh.b4.h4", Nhan: "Bước 4 — tên", Mac: "Sửa và chụp lại"},
					{Khoa: "quytrinh.b4.chu", Nhan: "Bước 4 — mô tả", Dai: true,
						Mac: "Em làm đúng quy trình của loại hỏng đó, xong chụp lại chỗ đã làm và gửi ảnh cho anh/chị xem. Bảo hành ghi trên đơn, tính từ ngày giao."},
					{Khoa: "quytrinh.b5.n", Nhan: "Bước 5 — số", Mac: "BƯỚC 5"},
					{Khoa: "quytrinh.b5.h4", Nhan: "Bước 5 — tên", Mac: "Cân lại rồi mới giao"},
					// {nguong} là ngưỡng tăng khối lượng, sửa số ở mục Ngưỡng.
					{Khoa: "quytrinh.b5.chu", Nhan: "Bước 5 — mô tả ({nguong} là mức tăng cân, sửa chữ đó ở nhóm Mức cân bên trên)", Dai: true,
						Mac: "Trước khi động vào, em cân vợt trên cân điện tử và ghi số vào đơn. Sửa xong cân lại, ghi số thứ hai. Cả hai số nằm trên phiếu giao. Chênh quá **{nguong}** thì em không lấy tiền công — đây là con số anh/chị kiểm tra lại được bằng một cái cân nhà bếp."},
				},
			},
			{
				Ten: "Hai đường đưa vợt tới trạm",
				Muc: []MucND{
					{Khoa: "quytrinh.duong.nhan", Nhan: "Nhãn nhỏ", Mac: "Đưa vợt tới trạm"},
					{Khoa: "quytrinh.duong.h2", Nhan: "Tiêu đề mục",
						Mac:   "Hai đường, chọn đường nào cũng khám như nhau",
						MacOn: "Đặt trên web rồi gửi đồ tới — hoặc mang tới tận nơi"},
					{Khoa: "quytrinh.duong.den-h3", Nhan: "Đường 1 — tiêu đề", Mac: "Đến tận nơi, khám tại chỗ"},
					{Khoa: "quytrinh.duong.den-chu", Nhan: "Đường 1 — mô tả", Dai: true,
						Mac: "Anh/chị mang vợt tới trạm thì khám ngay trước mặt: gõ mặt, soi mép, chỉ tận tay chỗ hỏng nằm ở đâu. Có giá luôn trong buổi, khỏi phải chờ. Nhắn trước một câu để em xếp lịch, đỡ tới lúc thợ đang bận tay giữa một ca khác."},
					{Khoa: "quytrinh.duong.gui-h3", Nhan: "Đường 2 — tiêu đề", Mac: "Gửi vợt từ xa"},
					{Khoa: "quytrinh.duong.gui-chu", Nhan: "Đường 2 — mô tả", Dai: true,
						Mac:   "Bọc vợt bằng xốp hơi, cho vào hộp cứng. Ghi mã yêu cầu lên ngoài hộp. Vợt về tới nơi là em khám và báo giá trong ngày, kèm ảnh chụp chỗ hỏng để anh/chị thấy đúng thứ em thấy.",
						MacOn: "Đặt đơn trên web trước để lấy mã. Bọc vợt bằng xốp hơi, cho vào hộp cứng, **ghi mã đơn lên ngoài hộp** — không có mã thì kiện tới nơi trạm không biết của ai. Vợt về tới nơi là em khám và báo giá trong ngày, kèm ảnh chụp chỗ hỏng."},
				},
			},
			{
				Ten: "Phí vận chuyển",
				Muc: []MucND{
					{Khoa: "quytrinh.ship.h3", Nhan: "Tiêu đề thẻ", Mac: "Phí vận chuyển"},
					{Khoa: "quytrinh.ship.chu-1", Nhan: "Đoạn 1 — {nguong} là mức miễn phí chiều về, sửa số đó ở mục Ngưỡng", Dai: true,
						Mac: "Anh/chị chịu phí gửi vợt tới trạm. Chiều gửi trả cũng vậy — trừ khi đơn từ `{nguong}` trở lên, lúc đó trạm chịu chiều về."},
					{Khoa: "quytrinh.ship.chu-2", Nhan: "Đoạn 2", Dai: true,
						Mac:   "Mang tới tận nơi thì không có khoản này. Tiền khám vợt và tư vấn không tính, kể cả khi khám xong anh/chị quyết định không sửa.",
						MacOn: "Tiền khám vợt và tư vấn không tính, kể cả khi khám xong anh/chị quyết định không sửa — lúc đó trạm chỉ thu đúng phí gửi vợt về."},
				},
			},
			{
				Ten: "Thẻ bên phải",
				Muc: []MucND{
					{Khoa: "quytrinh.theo.h3", Nhan: "Thẻ 1 — tiêu đề", Mac: "Theo dõi vợt đang ở bước nào"},
					{Khoa: "quytrinh.theo.chu", Nhan: "Thẻ 1 — mô tả", Dai: true,
						Mac: "Mỗi đơn có một mã dạng `TV-2609-001`. Gõ mã đơn và 4 số cuối điện thoại là thấy vợt đang ở bước nào, hẹn trả ngày nào, đã đưa bao nhiêu tiền, còn thiếu bao nhiêu."},
					{Khoa: "quytrinh.chua.h3", Nhan: "Thẻ 2 — tiêu đề", Mac: "Chưa gửi vợt"},
					{Khoa: "quytrinh.chua.chu", Nhan: "Thẻ 2 — mô tả", Dai: true,
						Mac: "Gửi ảnh chỗ hỏng trước, em xem rồi nhận xét trong ngày — chưa cần gửi vợt đi vội."},
				},
			},
		},
	},
	{
		Ma:   "vechungtoi",
		Ten:  "Trang Về chúng tôi",
		MoTa: "Trạm ở đâu, mở cửa lúc nào, và trong trạm trông ra sao. Trang giới thiệu nói trạm LÀM GÌ, trang này nói trạm CÓ THẬT — hai câu hỏi khác nhau nên để hai trang.",
		Nhom: []NhomND{
			{
				Ten: "Đầu trang",
				Muc: []MucND{
					{Khoa: "vechungtoi.nhan", Nhan: "Nhãn nhỏ", Mac: "Về chúng tôi"},
					{Khoa: "vechungtoi.h1", Nhan: "Tiêu đề",
						Mac:   "Trạm có địa chỉ thật, anh/chị ghé xem được",
						MacOn: "Trạm có địa chỉ thật, không phải một tài khoản trên mạng"},
					{Khoa: "vechungtoi.dan", Nhan: "Đoạn dẫn", Dai: true,
						Mac: "Sửa vợt là nghề gửi đồ đi rồi chờ. Nên trước khi anh/chị gửi cây vợt vài triệu cho một người lạ trên mạng, đây là chỗ trạm ngồi và những gì có trong đó."},
				},
			},
			{
				Ten: "Trạm ở đâu",
				Muc: []MucND{
					{Khoa: "vechungtoi.o.nhan", Nhan: "Nhãn nhỏ", Mac: "Địa chỉ", MacOn: "Địa chỉ nhận hàng"},
					{Khoa: "vechungtoi.o.h2", Nhan: "Tiêu đề mục", Mac: "Trạm ở đây", MacOn: "Gửi vợt về đây"},
					{Khoa: "vechungtoi.o.chua-co", Nhan: "Câu hiện khi chưa điền địa chỉ ở /qt/lien-he", Dai: true,
						Mac:   "Địa chỉ đang cập nhật — anh/chị nhắn cho trạm để lấy chỉ đường.",
						MacOn: "Địa chỉ nhận hàng đang cập nhật — anh/chị nhắn cho trạm để biết gửi vợt tới đâu."},
					{Khoa: "vechungtoi.o.gio-nhan", Nhan: "Nhãn dòng giờ mở cửa", Mac: "Mở cửa", MacOn: "Giờ nhận hàng"},
					{Khoa: "vechungtoi.o.chi-duong", Nhan: "Nút mở bản đồ", Mac: "Chỉ đường trên Google Maps"},
					{Khoa: "vechungtoi.o.chu", Nhan: "Câu dưới địa chỉ", Dai: true,
						Mac:   "Ghé trực tiếp thì khám tại chỗ, anh/chị đứng xem thợ mở vợt luôn. Ở xa thì gửi chuyển phát cũng được — quy trình y hệt, chỉ khác là khám qua ảnh trước.",
						MacOn: "Đây cũng là địa chỉ anh/chị gửi đồ tới — ghi kèm tên và số điện thoại của trạm ở mục liên hệ. Muốn ghé xem tận nơi thì cứ ghé, nhắn trước một câu để trạm có người ở nhà."},
				},
			},
			{
				Ten: "Dải ảnh trạm",
				Muc: []MucND{
					{Khoa: "vechungtoi.anh.nhan", Nhan: "Nhãn nhỏ", Mac: "Trong trạm"},
					{Khoa: "vechungtoi.anh.h2", Nhan: "Tiêu đề mục", Mac: "Chỗ làm việc, chụp nguyên trạng"},
					{Khoa: "vechungtoi.anh.chu", Nhan: "Câu dưới tiêu đề", Dai: true,
						Mac: "Ảnh chụp bàn làm việc thật, không dọn. Up ảnh ở trang quản trị Ảnh trang chủ rồi bấm chuyển sang Về chúng tôi."},
					{Khoa: "vechungtoi.anh.chua-co", Nhan: "Câu hiện khi chưa có ảnh nào", Dai: true,
						Mac: "Ảnh trạm đang chụp, sẽ đưa lên trong ít ngày tới."},
				},
			},
			{
				// Khối chào hàng duy nhất của trang này. Mọi câu ở đây phải
				// chỉ được ra bằng chứng nằm sẵn trên site — hồ sơ ca, ngưỡng
				// cân ở /gioi-thieu, hai ca từ chối ở /dich-vu, dòng vệ sinh
				// giá 0 ở bảng giá. Đừng thêm số năm kinh nghiệm hay số khách
				// vào đây: /gioi-thieu đang nói thẳng "trạm mới mở", khách bấm
				// một cái là thấy hai trang cãi nhau.
				Ten: "Vì sao chọn trạm",
				Muc: []MucND{
					{Khoa: "vechungtoi.vs.nhan", Nhan: "Nhãn nhỏ", Mac: "Vì sao chọn trạm"},
					{Khoa: "vechungtoi.vs.h2", Nhan: "Tiêu đề mục",
						Mac: "Trạm nhỏ, nhưng làm nghề đến nơi đến chốn"},
					{Khoa: "vechungtoi.vs.dan", Nhan: "Đoạn dẫn", Dai: true,
						Mac:   "Sửa vợt ở Việt Nam phần lớn vẫn là nghề truyền miệng: mang tới, để đấy, vài hôm sau nhận về, hỏng chỗ nào cũng chẳng ai ghi lại. Trạm làm khác hẳn. Và cái khác ấy không nằm ở lời quảng cáo — nó nằm trong bốn thứ dưới đây, thứ nào anh/chị cũng tự kiểm tra được.",
						MacOn: "Sửa vợt ở Việt Nam phần lớn vẫn là nghề truyền miệng: gửi đi, để đấy, vài hôm sau nhận về, hỏng chỗ nào cũng chẳng ai ghi lại. Trạm làm khác hẳn. Và cái khác ấy không nằm ở lời quảng cáo — nó nằm trong bốn thứ dưới đây, thứ nào anh/chị cũng tự kiểm tra được."},

					{Khoa: "vechungtoi.vs.1-ten", Nhan: "Thẻ 1 — tên",
						Mac: "Mỗi ca một hồ sơ, không làm theo trí nhớ"},
					{Khoa: "vechungtoi.vs.1-chu", Nhan: "Thẻ 1 — mô tả", Dai: true,
						Mac: "Mỗi loại hỏng có một quy trình viết sẵn, làm đúng từng bước chứ không tùy tay. Cân trước, cân sau, ảnh trước, ảnh sau đều lưu vào đơn. Ca nào làm chưa đạt thì nằm lại trong hồ sơ để lần sau không lặp lại."},

					{Khoa: "vechungtoi.vs.2-ten", Nhan: "Thẻ 2 — tên",
						Mac: "Vượt ngưỡng cân là không lấy tiền công"},
					{Khoa: "vechungtoi.vs.2-chu", Nhan: "Thẻ 2 — mô tả", Dai: true,
						Mac: "Vợt nặng thêm là một cây vợt khác, đánh không còn quen tay. Nên trạm tự đặt trần tăng cân và tự phạt mình nếu vượt. Con số ấy ghi ở [trang giới thiệu](/gioi-thieu), và anh/chị soi được bằng đúng một cái cân nhà bếp."},

					{Khoa: "vechungtoi.vs.3-ten", Nhan: "Thẻ 3 — tên",
						Mac: "Có ca trạm không nhận, dù anh/chị trả tiền"},
					{Khoa: "vechungtoi.vs.3-chu", Nhan: "Thẻ 3 — mô tả", Dai: true,
						Mac: "Nứt ở tâm mặt vợt, hay lõi tổ ong đã sập, thì trạm nói không và nói rõ vì sao. Làm cho nó trông lành lặn thì vẫn làm được — nhưng cây vợt ấy gãy giữa trận. Tiền công đó trạm không lấy."},

					{Khoa: "vechungtoi.vs.4-ten", Nhan: "Thẻ 4 — tên",
						Mac: "Nhắn tin là gặp đúng người cầm cây vợt"},
					{Khoa: "vechungtoi.vs.4-chu", Nhan: "Thẻ 4 — mô tả", Dai: true,
						Mac: "Không tổng đài, không nhân viên đọc kịch bản. Người trả lời tin nhắn cũng là người mở cây vợt ra, nên anh/chị hỏi sâu tới đâu cũng có câu trả lời thật tới đó."},

					{Khoa: "vechungtoi.vs.chot", Nhan: "Câu chốt dưới bốn thẻ", Dai: true,
						Mac:   "Và ca nào trạm cũng vệ sinh vợt miễn phí trước khi trả — cây vợt về tay anh/chị phải sạch hơn lúc mang đến. Gửi ảnh chỗ hỏng là trạm xem rồi trả lời có nhận được hay không, trước cả chuyện giá.",
						MacOn: "Và ca nào trạm cũng vệ sinh vợt miễn phí trước khi gửi về — cây vợt về tay anh/chị phải sạch hơn lúc gửi đi. Gửi ảnh chỗ hỏng là trạm xem rồi trả lời có nhận được hay không, trước cả chuyện giá."},
					{Khoa: "vechungtoi.vs.nut", Nhan: "Nút sang trang giới thiệu", Mac: "Đọc cách trạm làm việc"},
				},
			},
			{
				Ten: "Chốt trang",
				Muc: []MucND{
					{Khoa: "vechungtoi.chot.h2", Nhan: "Tiêu đề", Mac: "Xem thêm"},
					{Khoa: "vechungtoi.chot.chu", Nhan: "Câu dẫn", Dai: true,
						Mac: "Muốn biết trạm làm việc thế nào thì đọc [trang giới thiệu](/gioi-thieu); muốn biết giá thì gửi ảnh chỗ hỏng, trạm xem rồi báo."},
				},
			},
		},
	},
	{
		Ma:    "app",
		Ten:   "App trên điện thoại",
		Rieng: true, // sửa ở /qt/app, không hiện tab ở /qt/noi-dung
		MoTa:  "Màn hình hiện ra khi khách mở app từ icon trên màn hình chính, cộng lời mời cài hiện ở mọi trang web. App chỉ có việc làm được với cây vợt — người bấm icon là đang muốn làm gì đó, không phải ngồi đọc.",
		Nhom: []NhomND{
			{
				Ten: "Màn hình app",
				Muc: []MucND{
					{Khoa: "app.bia.chi-duong", Nhan: "Bảng hiệu — chữ trên địa chỉ ở góc phải", Mac: "Chỉ đường"},
					{Khoa: "app.bia.nhan", Nhan: "Bảng hiệu — viên nhãn bên phải tên trạm, để trống thì ẩn hẳn",
						Mac: "Nhận sửa vợt & giày toàn quốc"},
					{Khoa: "app.luoi.h2", Nhan: "Tiêu đề trên lưới việc", Mac: "Trạm làm được gì"},
					{Khoa: "app.o-khac", Nhan: "Ô cuối lưới — tên", Mac: "Việc khác"},
					{Khoa: "app.o-khac-phu", Nhan: "Ô cuối lưới — dòng nhỏ", Mac: "Khám vợt"},
					{Khoa: "app.trong", Nhan: "Câu hiện khi chưa mở bán việc nào", Dai: true,
						Mac: "Trạm đang xếp lại bảng việc. Anh/chị cứ gửi ảnh chỗ hỏng, trạm xem rồi trả lời."},
				},
			},
			{
				Ten: "Banner đầu màn hình app",
				Muc: []MucND{
					// Xuống dòng ở đây là xuống dòng thật trên banner (CSS để
					// white-space: pre-line). Hai nghề là hai việc riêng, đọc
					// thành hai dòng thì mắt bắt được cả hai; nhồi một dòng thì
					// trình duyệt tự ngắt ở đâu tuỳ bề ngang máy — trên 390px nó
					// ngắt giữa "Thay đế giày" và "thể thao", thành ra dòng hai
					// là một mẩu cụt không có nghĩa.
					{Khoa: "app.banner.tieu-de", Nhan: "Dòng lớn — mỗi dòng gõ xuống là một dòng trên banner", Dai: true,
						Mac: "Sửa vợt Pickleball\nThay đế giày thể thao"},
					{Khoa: "app.banner.chu", Nhan: "Dòng nhỏ dưới — màn hình thấp thì ẩn đi", Dai: true,
						Mac: "Khám xong báo giá — chốt rồi thợ mới làm."},
				},
			},
			{
				Ten: "Khuyến mãi — khối ở màn hình đầu app",
				Muc: []MucND{
					{Khoa: "app.km.h2", Nhan: "Tiêu đề khối", Mac: "Đang có"},
					{Khoa: "app.km.nhan", Nhan: "Huy hiệu bên trái — vài ký tự thôi", Mac: "MỚI"},
					{Khoa: "app.km.ten", Nhan: "Tên chương trình — ĐỂ TRỐNG là cả khối biến mất", Mac: ""},
					{Khoa: "app.km.han", Nhan: "Dòng nhỏ: hạn, điều kiện", Mac: ""},
				},
			},
			{
				Ten: "Giá — hiện trên ô việc và trong trang từng dịch vụ",
				Muc: []MucND{
					{Khoa: "app.gia.h2", Nhan: "Nhãn đứng trước con số", Mac: "Giá tham khảo"},
					{Khoa: "app.gia.tu", Nhan: "Chữ đứng trước con số", Mac: "từ"},
					{Khoa: "app.gia.den", Nhan: "Chữ nối hai đầu khoảng giá", Mac: "đến"},
					{Khoa: "app.gia.mien-phi", Nhan: "Việc trạm không thu tiền", Mac: "Miễn phí"},
					{Khoa: "app.gia.bao-rieng", Nhan: "Việc phải xem vợt mới có giá", Mac: "Xem vợt rồi báo"},
					{Khoa: "app.gia.ngan-hoi", Nhan: "Như trên nhưng in trong ô việc ở app — ô hẹp, để 2-3 chữ", Mac: "Báo sau"},
					{Khoa: "app.gia.tuy-the", Dai: true,
						Nhan: "Câu dưới lưới việc — chỉ hiện khi có việc chưa niêm yết giá cứng",
						Mac:  "Một số giá phụ thuộc vào tình trạng vợt. Khám xong trạm báo giá chính xác."},
				},
			},
			{
				Ten: "Cam kết của trạm — hiện ở /app/quy-trinh",
				Muc: []MucND{
					{Khoa: "app.cs.h2", Nhan: "Tiêu đề khối", Mac: "Trạm cam kết"},
					{Khoa: "app.cs.1", Nhan: "Cam kết 1 — tên", Mac: "Bảo hành sau sửa"},
					{Khoa: "app.cs.1-chu", Nhan: "Cam kết 1 — chi tiết", Dai: true,
						Mac: "Chỗ trạm vừa làm mà hỏng lại vì tay thợ thì trạm làm lại, không tính thêm tiền."},
					{Khoa: "app.cs.2", Nhan: "Cam kết 2 — tên", Mac: "Hẹn ngày trả rõ"},
					{Khoa: "app.cs.2-chu", Nhan: "Cam kết 2 — chi tiết", Dai: true,
						Mac: "Nhận vợt là có ngày lấy. Chậm thì trạm báo trước chứ không để anh/chị chờ rồi hỏi."},
					{Khoa: "app.cs.3", Nhan: "Cam kết 3 — tên", Mac: "Giao nhận tận nhà"},
					{Khoa: "app.cs.3-chu", Nhan: "Cam kết 3 — chi tiết", Dai: true,
						Mac:   "Không tiện mang tới thì trạm nhận và trả tại nhà. Đơn đủ lớn trạm chịu phí gửi trả.",
						MacOn: "Sửa xong trạm gửi về tận nhà. Đơn đủ lớn trạm chịu phí gửi trả."},
					{Khoa: "app.cs.4", Nhan: "Cam kết 4 — tên", Mac: "Không sửa được thì trả nguyên trạng"},
					{Khoa: "app.cs.4-chu", Nhan: "Cam kết 4 — chi tiết", Dai: true,
						Mac: "Khám xong thấy không cứu được, trạm gửi vợt về đúng như lúc nhận, không mất phí sửa."},
				},
			},
			{
				Ten: "Thanh dưới đáy — hiện ở cả app lẫn web trên điện thoại",
				Muc: []MucND{
					{Khoa: "app.nut.trang-chu", Nhan: "Nút 1 — về màn hình app", Mac: "Trang chủ"},
					{Khoa: "app.nut.tra-cuu", Nhan: "Nút 2 — tra cứu đơn", Mac: "Tra đơn"},
					{Khoa: "app.nut.lien-he", Nhan: "Nút giữa nhô lên — chữ hiện khi chưa điền số lẫn Zalo", Mac: "Liên hệ"},
					{Khoa: "app.lh.goi", Nhan: "Nút giữa — dòng gọi điện trong bảng bật lên", Mac: "Gọi điện"},
					{Khoa: "app.nut.zalo", Nhan: "Nút giữa — dòng Zalo trong bảng bật lên", Mac: "Zalo"},
					{Khoa: "app.nut.gui-anh", Nhan: "Nút 4, cũng là lối tắt khi giữ lâu vào icon app", Mac: "Khám vợt"},
					{Khoa: "app.nut.quy-trinh", Nhan: "Nút 5", Mac: "Quy trình"},
				},
			},
			{
				Ten: "Màn hình con của app — tra đơn, gửi ảnh",
				Muc: []MucND{
					{Khoa: "app.tra.luu-nhan", Nhan: "Thẻ đơn máy nhớ giùm — chữ đứng trước mã đơn", Mac: "Mở lại đơn "},
					{Khoa: "app.tra.quen", Nhan: "Nút bỏ đơn máy đang nhớ", Mac: "Quên đơn này"},
					{Khoa: "app.gui.them", Nhan: "Nếp gấp chứa mấy ô không bắt buộc ở màn gửi ảnh", Mac: "Thêm tên, hãng vợt, email"},
					{Khoa: "app.gui.chup", Nhan: "Nút mở máy ảnh ở màn gửi ảnh", Mac: "Chụp ảnh"},
					{Khoa: "app.gui.chup-phu", Nhan: "Chữ nhỏ dưới nút chụp ảnh", Mac: "Mở máy ảnh chụp ngay"},
					{Khoa: "app.gui.may", Nhan: "Nút lấy ảnh có sẵn trong máy", Mac: "Chọn từ máy"},
					{Khoa: "app.gui.may-phu", Nhan: "Chữ nhỏ dưới nút chọn từ máy", Mac: "Ảnh hoặc video đã chụp"},
					{Khoa: "app.nut.quay-lai", Nhan: "Mũi tên lùi ở góc trái — chữ cho người dùng máy đọc màn hình", Mac: "Quay lại"},
				},
			},
			{
				Ten: "Lời mời cài app",
				Muc: []MucND{
					{Khoa: "caiapp.tieu-de", Nhan: "Dòng đậm — {ten} là tên thương hiệu", Mac: "Cài {ten} vào máy"},
					{Khoa: "caiapp.chu", Nhan: "Dòng nhỏ dưới tiêu đề", Dai: true,
						Mac: "Một chạm là mở, không cần nhớ địa chỉ web."},
					{Khoa: "caiapp.nut", Nhan: "Nút đồng ý", Mac: "Cài"},
					{Khoa: "caiapp.bo-qua", Nhan: "Nút đóng (chỉ máy đọc màn hình nghe thấy)", Mac: "Để sau"},
					{Khoa: "caiapp.ios", Nhan: "Câu mở đầu cho iPhone — Safari không cho web tự cài", Dai: true,
						Mac: "iPhone phải thêm bằng tay, hai bước thôi:"},
					{Khoa: "caiapp.ios-b1", Nhan: "iPhone — bước 1 (biểu tượng Chia sẻ vẽ sẵn ở cuối dòng)", Mac: "Bấm nút Chia sẻ ở thanh dưới"},
					{Khoa: "caiapp.ios-b2", Nhan: "iPhone — bước 2", Mac: "Chọn Thêm vào Màn hình chính"},
				},
			},
		},
	},
	{
		Ma:   "gioithieu",
		Ten:  "Trang giới thiệu",
		MoTa: "Trạm là ai, vì sao cam kết cân vợt, và vì sao có ca không nhận.",
		Nhom: []NhomND{
			{
				Ten: "Đầu trang",
				Muc: []MucND{
					{Khoa: "gioithieu.nhan", Nhan: "Nhãn nhỏ", Mac: "Giới thiệu"},
					{Khoa: "gioithieu.h1", Nhan: "Tiêu đề lớn", Mac: "Một trạm nhỏ, làm ít việc nhưng làm cho ra việc"},
					{Khoa: "gioithieu.dan", Nhan: "Đoạn dẫn — {ten} là tên trạm", Dai: true,
						Mac: "{ten} là trạm sửa vợt pickleball. Trạm mới mở, và em nói thẳng điều đó thay vì gắn một con số kinh nghiệm không kiểm chứng được lên trang chủ."},
					{Khoa: "gioithieu.doan-1", Nhan: "Đoạn mở", Dai: true,
						Mac: "Cái em bù lại được là cách làm: mỗi loại hỏng có một quy trình viết sẵn, làm đúng từng bước, và mỗi ca đều lưu lại số cân trước, số cân sau, ảnh trước, ảnh sau. Ca nào làm hỏng thì nằm trong hồ sơ để lần sau không lặp lại."},
				},
			},
			{
				Ten: "Mục cân vợt",
				Muc: []MucND{
					{Khoa: "gioithieu.can.nhan", Nhan: "Nhãn nhỏ", Mac: "Con số duy nhất khách kiểm tra được"},
					{Khoa: "gioithieu.can.h2", Nhan: "Tiêu đề mục", Mac: "Vì sao cứ nhắc mãi chuyện cân vợt"},
					{Khoa: "gioithieu.can.chu-1", Nhan: "Đoạn 1", Dai: true,
						Mac: "Vợt pickleball là một cây vợt được cân đo rất kỹ ở nhà máy. Dán thêm một lớp keo, đắp thêm một mảng vá là đổi trọng lượng và đổi điểm cân bằng. Vợt nặng thêm 8–10g thì cảm giác đánh khác hẳn — không phải cây vợt anh/chị quen nữa."},
					{Khoa: "gioithieu.can.chu-2", Nhan: "Đoạn 2 — {nguong} là mức tăng cân, sửa chữ đó ở nhóm Mức cân của trang Dùng chung", Dai: true,
						Mac: "Nên mức trạm cam kết là **{nguong}**. Vượt ngưỡng, em không lấy tiền công. Đây là con số duy nhất trong nghề này mà khách kiểm tra được bằng một cái cân nhà bếp — nên nó là con số em chọn để chịu trách nhiệm."},
				},
			},
			{
				Ten: "Mục từ chối",
				Muc: []MucND{
					{Khoa: "gioithieu.tuchoi.nhan", Nhan: "Nhãn nhỏ", Mac: "Từ chối"},
					{Khoa: "gioithieu.tuchoi.h2", Nhan: "Tiêu đề mục", Mac: "Nói không cũng là một phần của nghề"},
					{Khoa: "gioithieu.tuchoi.chu-1", Nhan: "Đoạn 1", Dai: true,
						Mac: "Có những ca sửa được về mặt kỹ thuật nhưng không nên sửa. Nhận những ca đó thì em có tiền công, còn anh/chị có một cây vợt trông lành lặn mà đánh vài buổi lại hỏng — hoặc tệ hơn, gãy giữa trận."},
					{Khoa: "gioithieu.tuchoi.chu-2", Nhan: "Đoạn 2", Dai: true,
						Mac: "Nên khi anh/chị gửi ảnh, câu trả lời đầu tiên của em luôn là **có nhận hay không**, trước cả chuyện giá. Ca nào em không nhận, em nói rõ hỏng ở đâu và vì sao sửa xong vẫn không đánh được, để anh/chị còn quyết định mua cây mới cho đỡ tiếc tiền sửa."},
				},
			},
			{
				Ten: "Phiếu ngưỡng bên phải",
				Muc: []MucND{
					{Khoa: "gioithieu.phieu.dinh", Nhan: "Đỉnh phiếu", Mac: "Ngưỡng trạm"},
					{Khoa: "gioithieu.phieu.dinh-phu", Nhan: "Đỉnh phiếu — chữ nhỏ", Mac: "cam kết"},
					{Khoa: "gioithieu.phieu.d", Nhan: "Dòng 1 — nhãn", Mac: "Tăng cân"},
					{Khoa: "gioithieu.phieu.chot", Nhan: "Dòng chốt — nhãn", Mac: "Vượt ngưỡng"},
					{Khoa: "gioithieu.phieu.chot-so", Nhan: "Dòng chốt — số", Mac: "0 đ"},
				},
			},
			{
				Ten: "Thẻ thay đế giày",
				Muc: []MucND{
					{Khoa: "gioithieu.giay.h3", Nhan: "Tiêu đề thẻ", Mac: "Về việc thay đế giày"},
					{Khoa: "gioithieu.giay.chu-1", Nhan: "Đoạn 1", Dai: true,
						Mac: "Trạm có nhận thay đế giày, nhưng chỉ với giày đi lại và giày chạy bộ. Giày sân — cầu lông, pickleball, tennis — thì không."},
					{Khoa: "gioithieu.giay.chu-2", Nhan: "Đoạn 2", Dai: true,
						Mac: "Đế giày sân được thiết kế để trượt xoay đúng một mức nhất định. Thay bằng đế bám hơn thì chân dừng mà người còn xoay, và đó đúng là cơ chế gây đứt dây chằng chéo trước. Một đôi giày làm lại đế sai loại rẻ hơn một ca mổ, nhưng đắt hơn rất nhiều so với việc mua giày mới."},
					{Khoa: "gioithieu.xem.h3", Nhan: "Thẻ xem tiếp — tiêu đề", Mac: "Xem tiếp"},
				},
			},
		},
	},
	{
		Ma:   "baiviet",
		Ten:  "Trang bài viết",
		MoTa: "Danh sách bài. Tiêu đề và tóm tắt từng bài sửa ở mục Bài viết.",
		Nhom: []NhomND{
			{
				Ten: "Đầu trang",
				Muc: []MucND{
					{Khoa: "baiviet.nhan", Nhan: "Nhãn nhỏ", Mac: "Bài viết"},
					{Khoa: "baiviet.h1", Nhan: "Tiêu đề lớn", Mac: "Viết lại những gì em phải giải thích nhiều lần"},
					{Khoa: "baiviet.dan", Nhan: "Đoạn dẫn", Dai: true,
						Mac: "Phần lớn câu hỏi khách nhắn tới đều lặp lại: cây này còn cứu được không, sửa xong có nặng lên không, bao lâu thì lấy được. Trả lời một lần cho kỹ rồi để đây, ai cần thì đọc trước khi gửi vợt đi."},
					{Khoa: "baiviet.nut", Nhan: "Nút dưới mỗi bài", Mac: "Đọc bài"},
				},
			},
		},
	},
	{
		Ma:   "baivietmot",
		Ten:  "Trang một bài viết",
		MoTa: "Khung quanh nội dung bài. Thân bài sửa ở mục Bài viết.",
		Nhom: []NhomND{
			{
				Ten: "Đầu trang",
				Muc: []MucND{
					{Khoa: "baivietmot.quay-lai", Nhan: "Link quay lại", Mac: "← Tất cả bài viết"},
					{Khoa: "baivietmot.dang", Nhan: "Chữ trước ngày đăng", Mac: "Đăng"},
					{Khoa: "baivietmot.cap-nhat", Nhan: "Chữ trước ngày sửa", Mac: "· cập nhật"},
				},
			},
			{
				Ten: "Thẻ bên phải",
				Muc: []MucND{
					{Khoa: "baivietmot.gui.h3", Nhan: "Thẻ 1 — tiêu đề", Mac: "Gửi ảnh chỗ hỏng"},
					{Khoa: "baivietmot.gui.chu", Nhan: "Thẻ 1 — mô tả", Dai: true,
						Mac: "Đọc xong vẫn chưa chắc cây vợt của mình thuộc nhóm nào? Gửi ảnh, em xem rồi nói thẳng là sửa được hay không."},
					{Khoa: "baivietmot.dv.h3", Nhan: "Thẻ 2 — tiêu đề", Mac: "Trạm nhận việc gì"},
					{Khoa: "baivietmot.dv.chu", Nhan: "Thẻ 2 — mô tả, {so} là số dịch vụ", Dai: true,
						Mac: "{so} dịch vụ, mỗi việc có thời gian làm và thời hạn bảo hành riêng."},
					{Khoa: "baivietmot.tiep.h3", Nhan: "Thẻ 3 — tiêu đề", Mac: "Đọc tiếp"},
				},
			},
		},
	},
	{
		Ma:   "cua-hang",
		Ten:  "Cửa hàng",
		MoTa: "Trang đặt sửa từ xa. Chỉ hiện khi site chạy bản Online và đã có địa chỉ trạm.",
		Nhom: []NhomND{
			{
				Ten: "Mở đầu",
				Muc: []MucND{
					{Khoa: "cuahang.mo.tieu-de", Nhan: "Tiêu đề trang", Mac: "Đặt sửa online"},
					{Khoa: "cuahang.mo.dan", Nhan: "Câu dẫn", Dai: true,
						Mac: "Chọn món và việc cần làm, rồi gửi đồ tới. Khám xong trạm báo giá, anh/chị chốt thì thợ mới làm."},
				},
			},
			{
				Ten: "Các bước",
				Muc: []MucND{
					{Khoa: "cuahang.buoc.mon", Nhan: "Bước 1 — tên", Mac: "Anh/chị gửi gì?"},
					{Khoa: "cuahang.buoc.goi", Nhan: "Bước 2 — tên", Mac: "Cần làm gì?"},
					{Khoa: "cuahang.buoc.tinh-trang", Nhan: "Bước 3 — tên", Mac: "Tình trạng hiện tại"},
					{Khoa: "cuahang.buoc.nguoi-nhan", Nhan: "Bước 4 — tên", Mac: "Nhận lại ở đâu"},
					{Khoa: "cuahang.buoc.gia-tu", Nhan: "Nhãn đứng trước giá", Mac: "Giá từ"},
					{Khoa: "cuahang.buoc.bao-gia-rieng", Nhan: "Nhãn cho việc phải xem mới báo giá", Mac: "Xem món rồi báo giá"},
					{Khoa: "cuahang.buoc.gia-nhac", Nhan: "Câu nhắc dưới bảng giá", Dai: true,
						Mac: "Đây là giá khởi điểm. Giá chốt báo sau khi thợ khám, và anh/chị duyệt rồi thợ mới làm."},
					{Khoa: "cuahang.buoc.anh-nhac", Nhan: "Câu nhắc gửi ảnh", Dai: true,
						Mac: "Chụp giúp trạm chỗ hỏng, chụp gần và đủ sáng. Có ảnh thì thợ đoán được việc trước khi hàng tới."},
					{Khoa: "cuahang.buoc.nut", Nhan: "Nút gửi đơn", Mac: "Đặt sửa"},
				},
			},
			{
				Ten: "Nhãn từng ô trong biểu mẫu",
				Muc: []MucND{
					{Khoa: "cuahang.o.mon-vot", Nhan: "Chọn món — vợt", Mac: "Vợt pickleball"},
					{Khoa: "cuahang.o.mon-giay", Nhan: "Chọn món — giày", Mac: "Giày"},
					{Khoa: "cuahang.o.giay-hang", Nhan: "Ô hãng giày", Mac: "Hãng / đời giày"},
					{Khoa: "cuahang.o.giay-size", Nhan: "Ô size giày", Mac: "Size giày"},
					{Khoa: "cuahang.o.giay-kieu", Nhan: "Ô kiểu giày", Mac: "Kiểu giày"},
					{Khoa: "cuahang.o.chua-chon", Nhan: "Dòng đầu ô kiểu giày khi chưa chọn", Mac: "— chọn —"},
					{Khoa: "cuahang.o.giay-chay", Nhan: "Kiểu giày — lựa chọn 1", Mac: "Giày chạy bộ"},
					{Khoa: "cuahang.o.giay-di", Nhan: "Kiểu giày — lựa chọn 2", Mac: "Giày đi lại"},
					{Khoa: "cuahang.o.vot-hang", Nhan: "Ô hãng vợt", Mac: "Hãng / đời vợt"},
					{Khoa: "cuahang.o.gia-tri", Nhan: "Ô giá trị món", Mac: "Giá trị món (đ)"},
					{Khoa: "cuahang.o.tinh-trang", Nhan: "Ô mô tả chỗ hỏng", Mac: "Mô tả chỗ hỏng"},
					{Khoa: "cuahang.o.anh", Nhan: "Ô đính ảnh", Mac: "Ảnh chỗ hỏng"},
					{Khoa: "cuahang.o.ten", Nhan: "Ô tên người nhận", Mac: "Tên"},
					{Khoa: "cuahang.o.lien-he", Nhan: "Ô số điện thoại", Mac: "Số điện thoại / Zalo"},
					{Khoa: "cuahang.o.dia-chi", Nhan: "Ô địa chỉ nhận lại", Mac: "Địa chỉ nhận lại"},
				},
			},
			{
				Ten: "Màn kết — sau khi đặt xong",
				Muc: []MucND{
					{Khoa: "cuahang.xong.tieu-de", Nhan: "Tiêu đề", Mac: "Đã nhận đơn. Giờ gửi đồ tới nhé"},
					{Khoa: "cuahang.xong.ma-nhan", Nhan: "Nhãn trên ô mã đơn", Mac: "Mã đơn"},
					{Khoa: "cuahang.xong.ghi-ma", Nhan: "Câu nhắc ghi mã lên kiện", Dai: true,
						Mac: "**Ghi mã đơn lên kiện hàng** — hoặc kẹp một mẩu giấy có mã vào trong. Không có mã thì kiện tới nơi trạm không biết của ai."},
					{Khoa: "cuahang.xong.gui-toi", Nhan: "Tiêu đề khối địa chỉ", Mac: "Gửi tới"},
					{Khoa: "cuahang.xong.dong-goi-nhan", Nhan: "Tiêu đề khối đóng gói", Mac: "Đóng gói"},
					{Khoa: "cuahang.xong.dong-goi", Nhan: "Hướng dẫn đóng gói", Dai: true,
						Mac: "Bọc vợt bằng xốp hơi hoặc khăn dày, kỹ nhất ở cán và viền. Cho vào hộp cứng, chèn kín để món không xê dịch trong hộp. Giày thì buộc dây lại, nhét giấy vào mũi cho giữ phom."},
					{Khoa: "cuahang.xong.theo-doi", Nhan: "Tiêu đề khối tra cứu", Mac: "Theo dõi đơn"},
					{Khoa: "cuahang.xong.tra-cuu", Nhan: "Câu nhắc lưu link tra cứu", Dai: true,
						Mac: "Lưu lại đường dẫn này. Mọi cập nhật của đơn hiện ở đó, không cần đăng nhập."},
					{Khoa: "cuahang.xong.chep", Nhan: "Nút chép vào bộ nhớ tạm", Mac: "Chép"},
				},
			},
			{
				Ten: "Khi trạm chưa nhận món đó",
				Muc: []MucND{
					{Khoa: "cuahang.tuchoi.giay-kieu", Nhan: "Từ chối — kiểu giày", Dai: true,
						Mac: "Trạm mới thay đế cho giày chạy bộ và giày đi lại. Kiểu khác anh/chị nhắn trước để trạm xem ảnh đã, đừng gửi hàng đi vội."},
					{Khoa: "cuahang.tuchoi.duoi-nguong", Nhan: "Từ chối — món rẻ hơn hai chiều ship", Dai: true,
						Mac: "Món này rẻ hơn tiền ship hai chiều cộng công sửa. Gửi đi là anh/chị lỗ. Trạm chỉ nhận gửi từ"},
				},
			},
			{
				Ten: "Ô báo mã vận đơn",
				Muc: []MucND{
					{Khoa: "cuahang.vandon.tieu-de", Nhan: "Tiêu đề ô", Mac: "Đã gửi hàng rồi?"},
					{Khoa: "cuahang.vandon.dan", Nhan: "Câu dẫn", Dai: true,
						Mac: "Dán mã vận đơn vào đây để trạm biết kiện đang trên đường. Chưa gửi thì để trống cũng được."},
					{Khoa: "cuahang.vandon.o", Nhan: "Nhãn ô nhập", Mac: "Mã vận đơn"},
					{Khoa: "cuahang.vandon.nut", Nhan: "Nút lưu", Mac: "Lưu mã vận đơn"},
					{Khoa: "cuahang.vandon.ve", Nhan: "Nhãn mã vận đơn trạm gửi về", Mac: "Mã vận đơn gửi về"},
				},
			},
			{
				Ten: "Trả tiền sau khi sửa xong",
				Muc: []MucND{
					{Khoa: "cuahang.tra.tieu-de", Nhan: "Tiêu đề khối", Mac: "Thanh toán"},
					{Khoa: "cuahang.tra.free-ship", Nhan: "Báo được miễn phí gửi về", Dai: true,
						Mac: "Đơn này trạm chịu phí gửi trả, anh/chị chỉ trả tiền sửa."},
					{Khoa: "cuahang.tra.qr", Nhan: "Tiêu đề cách 1 — chuyển khoản", Mac: "Chuyển khoản"},
					{Khoa: "cuahang.tra.qr-nhac", Nhan: "Nhắc khi chuyển khoản", Dai: true,
						Mac: "Nội dung chuyển khoản phải có **mã đơn** ở trên. Ghi đúng thì tiền vào là trạm đối chiếu được ngay, không phải nhắn hỏi lại."},
					{Khoa: "cuahang.tra.cod", Nhan: "Tiêu đề cách 2 — trả khi nhận", Mac: "Trả khi nhận hàng"},
					{Khoa: "cuahang.tra.cod-nhac", Nhan: "Nhắc khi trả COD", Dai: true,
						Mac: "Trạm gửi hàng thu hộ, anh/chị trả tiền cho bên giao rồi mới nhận. Tiền ship về tính thêm theo bảng giá của hãng vận chuyển."},
					{Khoa: "cuahang.tra.k-ngan-hang", Nhan: "Bảng chuyển khoản — dòng ngân hàng", Mac: "Ngân hàng"},
					{Khoa: "cuahang.tra.k-so-tk", Nhan: "Bảng chuyển khoản — dòng số tài khoản", Mac: "Số tài khoản"},
					{Khoa: "cuahang.tra.k-chu-tk", Nhan: "Bảng chuyển khoản — dòng chủ tài khoản", Mac: "Chủ tài khoản"},
					{Khoa: "cuahang.tra.k-so-tien", Nhan: "Bảng chuyển khoản — dòng số tiền", Mac: "Số tiền"},
					{Khoa: "cuahang.tra.k-noi-dung", Nhan: "Bảng chuyển khoản — dòng nội dung", Mac: "Nội dung"},
					{Khoa: "cuahang.tra.nut-qr", Nhan: "Nút chọn chuyển khoản", Mac: "Tôi sẽ chuyển khoản"},
					{Khoa: "cuahang.tra.nut-cod", Nhan: "Nút chọn trả khi nhận", Mac: "Tôi trả khi nhận hàng"},
					{Khoa: "cuahang.tra.da-chon-qr", Nhan: "Câu xác nhận sau khi chọn chuyển khoản", Dai: true,
						Mac: "Anh/chị đã chọn: chuyển khoản."},
					{Khoa: "cuahang.tra.da-chon-cod", Nhan: "Câu xác nhận sau khi chọn trả khi nhận", Dai: true,
						Mac: "Anh/chị đã chọn: trả khi nhận hàng."},
				},
			},
			{
				Ten: "Khi cửa hàng đóng",
				Muc: []MucND{
					{Khoa: "cuahang.dong.tieu-de", Nhan: "Tiêu đề", Mac: "Tạm chưa nhận đơn online"},
					{Khoa: "cuahang.dong.than", Nhan: "Nội dung", Dai: true,
						Mac: "Anh/chị nhắn Zalo giúp trạm, hoặc gọi trực tiếp. Trạm vẫn nhận sửa bình thường."},
				},
			},
		},
	},
	{
		Ma:   "tracuu",
		Ten:  "Trang tra cứu",
		MoTa: "Khách gõ mã để xem vợt đang ở bước nào. Chữ trạng thái đơn (Đang sửa, Đã xong…) sửa ở chỗ khác.",
		Nhom: []NhomND{
			{
				Ten: "Đầu trang và ô nhập",
				Muc: []MucND{
					{Khoa: "tracuu.nhan", Nhan: "Nhãn nhỏ", Mac: "Tra cứu"},
					{Khoa: "tracuu.h1", Nhan: "Tiêu đề lớn", Mac: "Vợt của tôi đang ở bước nào?"},
					{Khoa: "tracuu.dan", Nhan: "Đoạn dẫn", Dai: true,
						Mac: "Gõ mã ghi trên phiếu — hoặc mã yêu cầu hiện ra sau khi anh/chị gửi ảnh ở trang chủ — cùng 4 số cuối điện thoại đã để lại. Cần cả hai: chỉ mỗi mã thì người khác cũng dò ra đơn của anh/chị."},
					{Khoa: "tracuu.ma-nhan", Nhan: "Nhãn ô mã", Mac: "Mã đơn hoặc mã yêu cầu"},
					{Khoa: "tracuu.ma-goi-y", Nhan: "Chữ mờ trong ô mã", Mac: "TV-2609-001 · 2026-09-03-141522-K7P"},
					{Khoa: "tracuu.dt-nhan", Nhan: "Nhãn ô điện thoại", Mac: "4 số cuối điện thoại"},
					{Khoa: "tracuu.nut", Nhan: "Nút", Mac: "Tra cứu"},
				},
			},
			{
				Ten: "Khi mới gửi ảnh, chưa thành đơn",
				Muc: []MucND{
					{Khoa: "tracuu.yc.tieu", Nhan: "Chữ trước mã yêu cầu", Mac: "Yêu cầu"},
					{Khoa: "tracuu.yc.ngay", Nhan: "Chữ trước ngày gửi", Mac: "· gửi ngày"},
					{Khoa: "tracuu.yc.xong-chip", Nhan: "Nhãn khi trạm đã xem", Mac: "Trạm đã xem"},
					{Khoa: "tracuu.yc.xong-chu", Nhan: "Câu khi trạm đã xem", Dai: true,
						Mac: "Trạm đã xem ảnh và liên hệ lại theo số anh/chị để lại. Chưa thấy tin nhắn thì nhắn cho trạm kèm mã này."},
					{Khoa: "tracuu.yc.cho-chip", Nhan: "Nhãn khi đang chờ", Mac: "Đang chờ báo giá"},
					{Khoa: "tracuu.yc.cho-chu", Nhan: "Câu khi đang chờ", Dai: true,
						Mac: "Trạm đã nhận ảnh, đang xem để báo giá. Có giá rồi trạm nhắn lại trong ngày — chốt giá xong mới bắt tay vào làm."},
					{Khoa: "tracuu.yc.k-ten", Nhan: "Nhãn dòng Tên", Mac: "Tên"},
					{Khoa: "tracuu.yc.k-mota", Nhan: "Nhãn dòng mô tả", Mac: "Anh/chị mô tả"},
					{Khoa: "tracuu.yc.k-tep", Nhan: "Nhãn dòng ảnh", Mac: "Ảnh/video"},
					{Khoa: "tracuu.yc.tep-dv", Nhan: "Đơn vị đếm tệp", Mac: "tệp"},
					{Khoa: "tracuu.yc.duoi", Nhan: "Câu dưới khối yêu cầu", Dai: true,
						Mac: "Khi trạm dựng yêu cầu này thành đơn, anh/chị vẫn tra bằng đúng mã trên — lúc đó trang này hiện thêm tiền và ngày hẹn. [Liên hệ trạm](/lien-he) nếu cần gấp."},
				},
			},
			{
				Ten: "Khi đã thành đơn",
				Muc: []MucND{
					{Khoa: "tracuu.don.tieu", Nhan: "Chữ trước mã đơn", Mac: "Đơn"},
					{Khoa: "tracuu.don.ngay", Nhan: "Chữ trước ngày nhận", Mac: "· nhận ngày"},
					{Khoa: "tracuu.k-vot", Nhan: "Nhãn dòng Vợt", Mac: "Vợt"},
					{Khoa: "tracuu.don.k-chan", Nhan: "Nhãn dòng thợ ghi nhận", Mac: "Thợ ghi nhận"},
					{Khoa: "tracuu.don.k-hen", Nhan: "Nhãn dòng hẹn trả", Mac: "Hẹn trả"},
					{Khoa: "tracuu.don.k-bh", Nhan: "Nhãn dòng bảo hành", Mac: "Bảo hành đến"},
					{Khoa: "tracuu.don.tu-yc-1", Nhan: "Câu nối yêu cầu — vế trước mã", Mac: "Dựng từ yêu cầu"},
					{Khoa: "tracuu.don.tu-yc-2", Nhan: "Câu nối yêu cầu — vế sau mã", Mac: "anh/chị gửi ngày"},
					{Khoa: "tracuu.don.dien-bien", Nhan: "Tiêu đề khối diễn biến", Mac: "Diễn biến"},
				},
			},
			{
				Ten: "Khối tiền",
				Muc: []MucND{
					{Khoa: "tracuu.tien.h3", Nhan: "Tiêu đề khối", Mac: "Tiền"},
					{Khoa: "tracuu.tien.tong", Nhan: "Dòng tổng", Mac: "Tổng"},
					{Khoa: "tracuu.tien.da-thu", Nhan: "Dòng đã đưa", Mac: "Đã đưa"},
					{Khoa: "tracuu.tien.con", Nhan: "Dòng còn lại", Mac: "Còn lại"},
					{Khoa: "tracuu.sai.h3", Nhan: "Thẻ báo sai — tiêu đề", Mac: "Số nào sai?"},
					{Khoa: "tracuu.sai.chu", Nhan: "Thẻ báo sai — mô tả", Dai: true,
						Mac: "Đơn này ghi gì thì trạm chịu trách nhiệm đúng như vậy. Thấy lệch — tiền, ngày hẹn, hạng mục đã chốt — nhắn cho trạm kèm mã đơn."},
				},
			},
			{
				Ten: "Khối ảnh vợt",
				Muc: []MucND{
					{Khoa: "tracuu.anh.h3", Nhan: "Tiêu đề khối", Mac: "Ảnh vợt"},
					{Khoa: "tracuu.anh.truoc", Nhan: "Nhãn hàng ảnh trước", Mac: "Lúc trạm nhận vợt"},
					{Khoa: "tracuu.anh.sau", Nhan: "Nhãn hàng ảnh sau", Mac: "Sau khi sửa"},
					{Khoa: "tracuu.anh.alt-truoc", Nhan: "Chữ mô tả ảnh trước (cho trình đọc màn hình)", Mac: "Vợt lúc trạm nhận"},
					{Khoa: "tracuu.anh.alt-sau", Nhan: "Chữ mô tả ảnh sau (cho trình đọc màn hình)", Mac: "Vợt sau khi sửa"},
					{Khoa: "tracuu.anh.chu", Nhan: "Câu dưới khối ảnh", Dai: true,
						Mac: "Ảnh do trạm chụp tại bàn, bấm vào xem cỡ lớn. Chưa thấy ảnh sau nghĩa là vợt còn đang làm. Thấy ảnh không khớp với vợt của mình thì nhắn ngay kèm mã đơn."},
				},
			},
		},
	},
	{
		Ma:   "lienhe",
		Ten:  "Trang liên hệ",
		MoTa: "Số điện thoại, Zalo, địa chỉ lấy từ mục Liên hệ trong cài đặt — ở đây chỉ sửa chữ.",
		Nhom: []NhomND{
			{
				Ten: "Đầu trang",
				Muc: []MucND{
					{Khoa: "lienhe.nhan", Nhan: "Nhãn nhỏ", Mac: "Liên hệ"},
					{Khoa: "lienhe.h1", Nhan: "Tiêu đề lớn", Mac: "Nhắn thẳng cho trạm"},
					{Khoa: "lienhe.dan", Nhan: "Đoạn dẫn", Dai: true,
						Mac: "Có ảnh chỗ hỏng thì em trả lời nhanh và chắc hơn nhiều — sửa được hay không, và hết bao nhiêu."},
				},
			},
			{
				Ten: "Khối form",
				Muc: []MucND{
					{Khoa: "lienhe.gui.nhan", Nhan: "Nhãn nhỏ", Mac: "Gửi yêu cầu"},
					{Khoa: "lienhe.gui.h2", Nhan: "Tiêu đề mục", Mac: "Gửi ảnh chỗ hỏng, em xem rồi nhận xét"},
					{Khoa: "lienhe.gui.dan", Nhan: "Đoạn dẫn trên form", Dai: true,
						Mac: "Vợt hay giày đều được. Chụp rõ chỗ hỏng, quay thêm 10–15 giây nếu vợt có tiếng lạ khi đánh. Em trả lời trong ngày là cây này thuộc việc gì và mất mấy ngày; con số thì đợi khám tận tay mới có."},
					{Khoa: "lienhe.rieng-tu", Nhan: "Câu dưới form", Dai: true,
						Mac: "Thông tin anh/chị gửi chỉ dùng để liên hệ về đúng cây vợt này. Chi tiết ở [chính sách bảo mật](/chinh-sach)."},
				},
			},
			{
				Ten: "Thẻ liên hệ trực tiếp",
				Muc: []MucND{
					{Khoa: "lienhe.lh.h3", Nhan: "Tiêu đề thẻ", Mac: "Liên hệ trực tiếp"},
					{Khoa: "lienhe.lh.dt", Nhan: "Nhãn điện thoại", Mac: "Điện thoại"},
					{Khoa: "lienhe.lh.zalo", Nhan: "Nhãn Zalo", Mac: "Zalo"},
					{Khoa: "lienhe.lh.email", Nhan: "Nhãn email", Mac: "Email"},
					{Khoa: "lienhe.lh.mxh", Nhan: "Nhãn hàng mạng xã hội", Mac: "Mạng xã hội"},
					{Khoa: "lienhe.lh.dia-chi", Nhan: "Nhãn địa chỉ", Mac: "Địa chỉ", MacOn: "Gửi vợt tới"},
					{Khoa: "lienhe.lh.gio", Nhan: "Nhãn giờ làm việc", Mac: "Giờ làm việc", MacOn: "Giờ nhận hàng"},
					{Khoa: "lienhe.trong", Nhan: "Câu hiện khi chưa điền thông tin liên hệ nào", Dai: true,
						Mac: "Số điện thoại và địa chỉ trạm đang cập nhật. Trong lúc chờ, anh/chị để lại số ở form bên — em gọi lại."},
				},
			},
			{
				Ten: "Thẻ đơn đang sửa",
				Muc: []MucND{
					{Khoa: "lienhe.don.h3", Nhan: "Tiêu đề thẻ", Mac: "Đơn đang sửa"},
					{Khoa: "lienhe.don.chu", Nhan: "Mô tả", Dai: true,
						Mac: "Tra bằng mã đơn trên phiếu và 4 số cuối điện thoại, không cần nhắn hỏi."},
				},
			},
		},
	},
	{
		Ma:   "cauhoi",
		Ten:  "Trang câu hỏi thường gặp",
		MoTa: "Chỉ phần khung của trang. Từng câu hỏi và câu trả lời nằm ở /qt/cau-hoi, thêm bớt được, không sửa ở đây.",
		Nhom: []NhomND{
			{
				Ten: "Đầu trang",
				Muc: []MucND{
					{Khoa: "cauhoi.nhan", Nhan: "Nhãn nhỏ", Mac: "Hỏi đáp"},
					{Khoa: "cauhoi.h1", Nhan: "Tiêu đề lớn", Mac: "Câu hỏi thường gặp"},
					{Khoa: "cauhoi.dan", Nhan: "Đoạn dẫn", Dai: true,
						Mac: "Mấy câu khách hay hỏi nhất, trả lời sẵn ở đây. Không thấy câu của mình thì nhắn, em trả lời trong ngày."},
				},
			},
			{
				Ten: "Khi chưa có câu nào",
				Muc: []MucND{
					{Khoa: "cauhoi.trong", Nhan: "Câu hiện khi danh sách còn rỗng", Dai: true,
						Mac: "Chưa có câu hỏi nào được đăng. Cứ nhắn thẳng, em trả lời rồi đưa lên đây cho người sau đỡ phải hỏi lại."},
				},
			},
			{
				Ten: "Thẻ chốt trang",
				Muc: []MucND{
					{Khoa: "cauhoi.chot.h3", Nhan: "Tiêu đề thẻ", Mac: "Còn câu chưa có ở đây?"},
					{Khoa: "cauhoi.chot.chu", Nhan: "Mô tả", Dai: true,
						Mac: "Gửi ảnh chỗ hỏng, em xem rồi nói thẳng làm được hay không và mất mấy ngày."},
					{Khoa: "cauhoi.chot.nut", Nhan: "Chữ trên nút", Mac: "Gửi ảnh hỏi trước"},
				},
			},
		},
	},
	{
		Ma:   "chinhsach",
		Ten:  "Trang chính sách",
		MoTa: "Chính sách bảo mật. Đây là chỗ khách và cơ quan quản lý đọc — sửa xong nên đọc lại cả trang một lượt.",
		Nhom: []NhomND{
			{
				Ten: "Đầu trang",
				Muc: []MucND{
					{Khoa: "chinhsach.nhan", Nhan: "Nhãn nhỏ", Mac: "Chính sách"},
					{Khoa: "chinhsach.h1", Nhan: "Tiêu đề lớn", Mac: "Chính sách bảo mật dữ liệu cá nhân"},
					{Khoa: "chinhsach.dan", Nhan: "Đoạn dẫn — {ten} là tên trạm", Dai: true,
						Mac: "Áp dụng cho trang web và hệ thống quản lý đơn của {ten}. Viết theo Luật Bảo vệ dữ liệu cá nhân 91/2025/QH15."},
				},
			},
			{
				Ten: "1. Thu thập những gì",
				Muc: []MucND{
					{Khoa: "chinhsach.m1.h3", Nhan: "Tiêu đề mục", Mac: "1. Trạm thu thập những gì"},
					{Khoa: "chinhsach.m1.y1", Nhan: "Gạch đầu dòng 1", Dai: true,
						Mac: "**Khi anh/chị gửi form:** tên, số điện thoại hoặc Zalo, hãng vợt, mô tả tình trạng vợt, và địa chỉ IP của lượt gửi."},
					{Khoa: "chinhsach.m1.y2", Nhan: "Gạch đầu dòng 2", Dai: true,
						Mac: "**Khi mở đơn sửa:** thêm số cân vợt trước và sau, ảnh vợt trước và sau, ngày hẹn trả, số tiền đã chốt và đã thu."},
					{Khoa: "chinhsach.m1.y3", Nhan: "Gạch đầu dòng 3", Dai: true,
						Mac: "Không thu thập ngày sinh, số căn cước, tài khoản ngân hàng. Không cài mã theo dõi quảng cáo."},
				},
			},
			{
				Ten: "2. Dùng để làm gì",
				Muc: []MucND{
					{Khoa: "chinhsach.m2.h3", Nhan: "Tiêu đề mục", Mac: "2. Dùng để làm gì"},
					{Khoa: "chinhsach.m2.chu", Nhan: "Nội dung", Dai: true,
						Mac: "Chỉ ba việc: liên hệ lại về đúng cây vợt anh/chị gửi; theo dõi đơn sửa và bảo hành; và làm sổ sách của trạm. Không bán, không cho thuê, không chia sẻ danh sách khách cho bên thứ ba nào. Địa chỉ IP chỉ dùng để chặn gửi form hàng loạt."},
				},
			},
			{
				Ten: "3. Ai xem được",
				Muc: []MucND{
					{Khoa: "chinhsach.m3.h3", Nhan: "Tiêu đề mục", Mac: "3. Ai xem được"},
					{Khoa: "chinhsach.m3.chu", Nhan: "Nội dung", Dai: true,
						Mac: "Chủ trạm xem được toàn bộ. Thợ chỉ xem được những đơn mình đang cầm và đơn chưa ai nhận. Mỗi người một tài khoản riêng, có mật khẩu riêng. Ảnh vợt chỉ mở được sau khi đăng nhập."},
				},
			},
			{
				Ten: "4. Lưu ở đâu, lưu bao lâu",
				Muc: []MucND{
					{Khoa: "chinhsach.m4.h3", Nhan: "Tiêu đề mục", Mac: "4. Lưu ở đâu, lưu bao lâu"},
					{Khoa: "chinhsach.m4.chu", Nhan: "Nội dung", Dai: true,
						Mac: "Dữ liệu nằm trên máy chủ do trạm thuê. Yêu cầu gửi từ form mà không thành đơn: xoá sau 12 tháng. Đơn sửa: giữ trong thời gian bảo hành cộng thêm 24 tháng, để có căn cứ khi anh/chị quay lại khiếu nại."},
				},
			},
			{
				Ten: "5. Quyền của anh/chị",
				Muc: []MucND{
					{Khoa: "chinhsach.m5.h3", Nhan: "Tiêu đề mục", Mac: "5. Quyền của anh/chị"},
					{Khoa: "chinhsach.m5.chu-1", Nhan: "Đoạn 1", Dai: true,
						Mac: "Anh/chị có quyền yêu cầu xem dữ liệu của mình, yêu cầu sửa hoặc xoá, rút lại đồng ý, và khiếu nại nếu trạm làm sai. Trang web không có chỗ tự sửa hay tự xoá — mọi thay đổi đều phải nhắn cho trạm theo thông tin ở trang [liên hệ](/lien-he), kèm mã đơn để em tìm đúng hồ sơ. Em xử lý trong vòng 72 giờ."},
					{Khoa: "chinhsach.m5.chu-2", Nhan: "Đoạn 2", Dai: true,
						Mac: "Làm qua người thay vì để tự bấm là có lý do: hồ sơ đơn sửa là căn cứ bảo hành cho chính anh/chị, xoá nhầm thì không dựng lại được. Muốn huỷ đơn hoặc xoá hồ sơ, nhắn cho trạm — em xác nhận lại một lần rồi mới làm, và nói rõ đơn đó mất căn cứ bảo hành kể từ lúc xoá."},
				},
			},
			{
				Ten: "6. Đường truyền",
				Muc: []MucND{
					{Khoa: "chinhsach.m6.h3", Nhan: "Tiêu đề mục", Mac: "6. Đường truyền"},
					{Khoa: "chinhsach.m6.chu", Nhan: "Nội dung", Dai: true,
						Mac: "Trang chạy trên tên miền riêng, có HTTPS: mọi thứ anh/chị gõ vào form đều được mã hoá trên đường truyền. Trạm chỉ hỏi tên, số điện thoại và mô tả cây vợt — đừng gửi thêm giấy tờ tuỳ thân hay số tài khoản, những thứ đó trạm không cần đến."},
				},
			},
		},
	},
	{
		Ma:   "dangnhap",
		Ten:  "Trang đăng nhập",
		MoTa: "Trang nội bộ cho thợ và chủ trạm. Không có đường dẫn nào trỏ tới đây — ai cần thì gõ thẳng /dang-nhap.",
		Nhom: []NhomND{
			{
				Ten: "Cả trang",
				Muc: []MucND{
					{Khoa: "dangnhap.nhan", Nhan: "Nhãn nhỏ", Mac: "Nội bộ"},
					{Khoa: "dangnhap.h1", Nhan: "Tiêu đề", Mac: "Đăng nhập"},
					{Khoa: "dangnhap.ten", Nhan: "Nhãn ô tên", Mac: "Tên đăng nhập"},
					{Khoa: "dangnhap.mat-khau", Nhan: "Nhãn ô mật khẩu", Mac: "Mật khẩu"},
					{Khoa: "dangnhap.nut", Nhan: "Nút", Mac: "Vào"},
					{Khoa: "dangnhap.chu", Nhan: "Câu dưới form", Dai: true,
						Mac: "Trang này dành cho thợ và chủ trạm. Khách tra cứu vợt của mình ở [trang tra cứu](/tra-cuu), không cần tài khoản."},
				},
			},
		},
	},
}
