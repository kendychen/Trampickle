package core

import "strings"

// BaSo là một con số đã xẻ làm ba: phần số, đơn vị đi liền sau, và phần chữ
// còn lại. Dải số ở đầu trang in ba mảnh này ở ba cỡ khác nhau.
type BaSo struct {
	So    string // "10–15"
	DonVi string // "g"
	Duoi  string // "tùy vợt"
}

// tachSo xẻ một chuỗi kiểu "10–15 g tùy vợt" hoặc "3 tháng" ra ba mảnh.
//
// Vì sao cần: ô thứ ba của dải số lấy chữ từ khoá chung.can.muc, mà khoá ấy
// còn dùng ở bốn trang khác nên không xẻ sẵn thành ba khoá được. In nguyên
// chuỗi ở cỡ 36px mono thì nó dài gấp ba hai ô bên cạnh và tụt hàng — đúng
// chỗ Kendy kêu "chưa được cân đối".
//
// Luật xẻ: chạy từ đầu chuỗi lấy hết ký tự số và dấu nối (– — - , . /) làm
// phần So. Từ kế tiếp làm DonVi. Phần còn lại làm Duoi. Chuỗi không bắt đầu
// bằng số thì trả về nguyên vẹn ở So, hai mảnh kia rỗng — để không bao giờ
// nuốt mất chữ của Kendy.
func tachSo(s string) BaSo {
	s = strings.TrimSpace(s)
	i := 0
	r := []rune(s)
	for i < len(r) && laSoHoacNoi(r[i]) {
		i++
	}
	if i == 0 {
		return BaSo{So: s}
	}
	b := BaSo{So: strings.TrimRight(string(r[:i]), " ")}
	con := strings.TrimSpace(string(r[i:]))
	if con == "" {
		return b
	}
	if k := strings.IndexRune(con, ' '); k >= 0 {
		b.DonVi, b.Duoi = con[:k], strings.TrimSpace(con[k+1:])
	} else {
		b.DonVi = con
	}
	return b
}

func laSoHoacNoi(c rune) bool {
	if c >= '0' && c <= '9' {
		return true
	}
	switch c {
	case '-', '–', '—', ',', '.', '/', '+':
		return true
	}
	return false
}
