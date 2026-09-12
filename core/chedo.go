// Mấy công tắc bộ mặt site, tất cả nằm trong data/giao-dien.yaml và đổi ở
// /qt/giao-dien:
//
//	che_do      — khách mang đồ tới xưởng, hay khách gửi đồ tới.
//	dv_noi_bat  — có tô nổi những việc đã tích ở /qt/dich-vu hay không.
//
// Chúng KHÔNG đổi màu, không đổi logo, không đổi bố cục. Gộp chuyện màu vào
// đây là dựng lại hệ theme đã cố tình xoá hồi làm Material 3.
//
// File chỉ có mấy khoá phẳng nên mỗi lần lưu là ghi lại TRỌN file từ một
// chuỗi cố định — không đọc-sửa-ghi. Đổi lấy: khoá nào cũng phải có mặt trong
// hàm ghi, quên một khoá là lần lưu sau xoá mất nó. Bù lại không bao giờ có
// cảnh file còn nửa cũ nửa mới.
//
// data/giao-dien.yaml từng giữ khoá "theme" của hệ giao diện cũ. Khoá ấy chết
// từ khi bỏ hệ theme; đọc file này phải bước qua nó mà không vấp.
package core

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strconv"
	"sync"

	"gopkg.in/yaml.v3"
)

const (
	CheDoTaiXuong = "tai_xuong"
	CheDoOnline   = "online"
)

type tepGiaoDien struct {
	CheDo string `yaml:"che_do"`
	// Con trỏ để phân biệt "chưa khai" với "khai false". Chưa khai thì bật:
	// file cũ không có khoá này, mà tích một việc xong không thấy gì đổi thì
	// Kendy sẽ đi tìm bug ở /qt/dich-vu chứ không nghĩ tới công tắc.
	DvNoiBat *bool `yaml:"dv_noi_bat"`
}

const dvNoiBatMacDinh = true

var (
	cheDoMu  sync.RWMutex
	cheDoNay = CheDoTaiXuong
	dvNoiBat = dvNoiBatMacDinh
)

func fileGiaoDien() string { return P("data/giao-dien.yaml") }

func CheDoHopLe(ma string) bool { return ma == CheDoTaiXuong || ma == CheDoOnline }

// NapCheDo đọc công tắc. Chưa có file không phải lỗi. File hỏng TRẢ lỗi nhưng
// vẫn để chế độ ở tại xưởng — chỗ gọi ghi nhật ký, web vẫn lên.
func NapCheDo() error {
	cheDoMu.Lock()
	cheDoNay, dvNoiBat = CheDoTaiXuong, dvNoiBatMacDinh
	cheDoMu.Unlock()

	b, err := os.ReadFile(fileGiaoDien())
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var t tepGiaoDien
	if err := yaml.Unmarshal(b, &t); err != nil {
		return fmt.Errorf("%s hỏng: %w", fileGiaoDien(), err)
	}
	cheDoMu.Lock()
	// Khoá trống hoặc giá trị lạ (kể cả file cũ chỉ có "theme"): chạy tại
	// xưởng. Không đoán.
	if CheDoHopLe(t.CheDo) {
		cheDoNay = t.CheDo
	}
	if t.DvNoiBat != nil {
		dvNoiBat = *t.DvNoiBat
	}
	cheDoMu.Unlock()
	return nil
}

func CheDoHienTai() string {
	cheDoMu.RLock()
	defer cheDoMu.RUnlock()
	return cheDoNay
}

func LaOnline() bool { return CheDoHienTai() == CheDoOnline }

// DvNoiBatBat cho biết khung nổi bật có được bật hay không. Tích ô ở
// /qt/dich-vu mà công tắc này tắt thì lưới trang chủ y như cũ.
func DvNoiBatBat() bool {
	cheDoMu.RLock()
	defer cheDoMu.RUnlock()
	return dvNoiBat
}

// ghiGiaoDien dựng lại trọn file. Mọi khoá phải có mặt ở đây.
func ghiGiaoDien(cheDo string, noiBat bool) error {
	noiDung := "# Bộ mặt site, đổi ở /qt/giao-dien.\n" +
		"# che_do — tai_xuong: khách mang đồ tới. online: khách gửi đồ tới.\n" +
		"# dv_noi_bat — tô nổi những việc đã tích ở /qt/dich-vu.\n\n" +
		"che_do: " + cheDo + "\n" +
		"dv_noi_bat: " + strconv.FormatBool(noiBat) + "\n"
	return ghiAtomic(fileGiaoDien(), []byte(noiDung))
}

// DatCheDo ghi ra đĩa TRƯỚC rồi mới đổi trong bộ nhớ. Ghi hỏng thì chế độ
// đang chạy không đổi — tránh cảnh admin thấy báo đã đổi mà lần khởi động sau
// quay về cũ.
func DatCheDo(ma string) error {
	if !CheDoHopLe(ma) {
		return fmt.Errorf("chế độ không hợp lệ: %q", ma)
	}
	if err := ghiGiaoDien(ma, DvNoiBatBat()); err != nil {
		return err
	}
	cheDoMu.Lock()
	cheDoNay = ma
	cheDoMu.Unlock()
	return nil
}

// DatDvNoiBat bật hay tắt khung nổi bật. Ghi trước, đổi bộ nhớ sau, y như
// DatCheDo và vì cùng một lý do.
func DatDvNoiBat(b bool) error {
	if err := ghiGiaoDien(CheDoHienTai(), b); err != nil {
		return err
	}
	cheDoMu.Lock()
	dvNoiBat = b
	cheDoMu.Unlock()
	return nil
}
