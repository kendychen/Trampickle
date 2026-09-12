package core

// Web Push: thông báo hiện trên màn hình khóa điện thoại, ngay cả khi app
// quản lý đang đóng.
//
// Vì sao tự viết thay vì lấy thư viện: cả trạm này không có dependency nào
// ngoài yaml.v3, và giữ được như thế là một tài sản — build lại sau hai năm
// vẫn chạy, không ai phải đi vá CVE của một thư viện push bỏ hoang. Việc cần
// làm cũng gọn: ký một JWT ES256 (VAPID, RFC 8292) rồi mã hóa thân tin bằng
// ECDH P-256 + HKDF-SHA256 + AES-128-GCM (RFC 8291 trên khung RFC 8188). Go
// 1.26 có sẵn crypto/ecdh, crypto/hkdf, crypto/ecdsa — đủ cả.
//
// Cặp khóa VAPID sinh một lần rồi giữ mãi trong data/bi-mat.yaml. Đổi khóa là
// mọi máy đã bật thông báo im lặng ngừng nhận, và không có gì báo cho họ biết.
//
// iPhone chỉ nhận push khi trang đã được "Thêm vào Màn hình chính" — Safari
// trong tab thường không có pushManager. Đó là luật của Apple, không phải lỗi
// ở đây.

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// --- Danh sách máy đã bật thông báo --------------------------------------

// PushMay là một "subscription" trình duyệt trả về sau khi Kendy đồng ý nhận
// thông báo. Mỗi máy một bản: điện thoại và máy tính là hai endpoint khác nhau.
type PushMay struct {
	Endpoint string `json:"endpoint"`
	// P256dh và Auth là base64url thô, đúng như trình duyệt đưa ra. Giữ nguyên
	// dạng chuỗi để file đọc được bằng mắt khi cần soi.
	P256dh string `json:"p256dh"`
	Auth   string `json:"auth"`
	May    string `json:"may"`
	Them   string `json:"them"`
	// LoiCuoi giữ lỗi gần nhất. Máy hỏng hẳn (404/410) thì bị xóa luôn, nên
	// trường này chỉ chứa lỗi tạm — mạng hỏng, dịch vụ push quá tải.
	LoiCuoi string `json:"loi_cuoi,omitempty"`
}

var pushMu sync.Mutex

func filePushMay() string { return P("data/push-may.json") }

func docPushMay() []PushMay {
	var ds []PushMay
	if b, err := os.ReadFile(filePushMay()); err == nil {
		json.Unmarshal(b, &ds)
	}
	return ds
}

func ghiPushMay(ds []PushMay) error {
	b, err := json.MarshalIndent(ds, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(filePushMay()), 0o755); err != nil {
		return err
	}
	return ghiAtomic(filePushMay(), b)
}

// DanhSachPushMay cho trang cài đặt đếm và liệt kê.
func DanhSachPushMay() []PushMay {
	pushMu.Lock()
	defer pushMu.Unlock()
	return docPushMay()
}

// ThemPushMay lưu một máy. Cùng endpoint thì GHI ĐÈ chứ không thêm dòng thứ
// hai: bấm lại nút "bật thông báo" là chuyện thường, và trình duyệt trả về
// đúng endpoint cũ — thêm dòng mới là mỗi tin gửi hai lần.
func ThemPushMay(m PushMay) error {
	m.Endpoint = strings.TrimSpace(m.Endpoint)
	m.P256dh = strings.TrimSpace(m.P256dh)
	m.Auth = strings.TrimSpace(m.Auth)
	if !strings.HasPrefix(m.Endpoint, "https://") || len(m.Endpoint) > 1000 {
		return errors.New("endpoint không hợp lệ")
	}
	if _, err := giaiB64(m.P256dh); err != nil {
		return errors.New("khóa p256dh không đọc được")
	}
	if _, err := giaiB64(m.Auth); err != nil {
		return errors.New("khóa auth không đọc được")
	}
	m.May = gonMotDong(m.May)
	m.Them = time.Now().Format("2006-01-02 15:04")

	pushMu.Lock()
	defer pushMu.Unlock()
	ds := docPushMay()
	for i := range ds {
		if ds[i].Endpoint == m.Endpoint {
			ds[i] = m
			return ghiPushMay(ds)
		}
	}
	// Chặn trên cho chắc: endpoint nào cũng của chính Kendy, mà một danh sách
	// phình lên nghìn dòng thì mỗi tin gửi nghìn request.
	if len(ds) >= 20 {
		ds = ds[len(ds)-19:]
	}
	return ghiPushMay(append(ds, m))
}

// XoaPushMay bỏ một máy khỏi danh sách. Gọi cả từ nút "tắt thông báo" và từ
// chỗ gửi khi dịch vụ push nói endpoint đã chết.
func XoaPushMay(endpoint string) error {
	pushMu.Lock()
	defer pushMu.Unlock()
	ds := docPushMay()
	con := make([]PushMay, 0, len(ds))
	for _, m := range ds {
		if m.Endpoint != endpoint {
			con = append(con, m)
		}
	}
	if len(con) == len(ds) {
		return nil
	}
	return ghiPushMay(con)
}

func ghiLoiPushMay(endpoint, loi string) {
	pushMu.Lock()
	defer pushMu.Unlock()
	ds := docPushMay()
	for i := range ds {
		if ds[i].Endpoint == endpoint {
			ds[i].LoiCuoi = loi
			ghiPushMay(ds)
			return
		}
	}
}

// --- Cặp khóa VAPID ------------------------------------------------------

var vapidMu sync.Mutex

func b64(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

// giaiB64 chịu cả hai lối viết base64url: có "=" đệm và không. Trình duyệt
// khác nhau trả khác nhau, và cả hai đều đúng.
func giaiB64(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, errors.New("rỗng")
	}
	if strings.HasSuffix(s, "=") {
		return base64.URLEncoding.DecodeString(s)
	}
	return base64.RawURLEncoding.DecodeString(s)
}

// KhoaVapidCong trả khóa công khai dạng base64url — thứ JS truyền vào
// applicationServerKey. Sinh cặp khóa nếu chưa có.
func KhoaVapidCong() (string, error) {
	pub, _, err := khoaVapid()
	return pub, err
}

// CoKhoaVapid — đã sinh khóa chưa, dùng cho trang cài đặt. KHÔNG sinh khóa:
// mở một trang không nên ghi file.
func CoKhoaVapid() bool {
	m := BiMatHienTai()
	return m.VapidPub != "" && m.VapidPriv != ""
}

// khoaVapid đọc cặp khóa, sinh và lưu nếu chưa có. Có khóa cửa riêng để hai
// request cùng lúc không sinh ra hai cặp khác nhau — cặp sau ghi đè cặp trước,
// và máy vừa đăng ký bằng cặp trước thành vô dụng.
func khoaVapid() (string, *ecdsa.PrivateKey, error) {
	vapidMu.Lock()
	defer vapidMu.Unlock()

	m := BiMatHienTai()
	if m.VapidPub != "" && m.VapidPriv != "" {
		k, err := docKhoaVapid(m.VapidPriv)
		if err != nil {
			return "", nil, err
		}
		return m.VapidPub, k, nil
	}

	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", nil, err
	}
	der, err := x509.MarshalPKCS8PrivateKey(k)
	if err != nil {
		return "", nil, err
	}
	cong, err := k.PublicKey.ECDH()
	if err != nil {
		return "", nil, err
	}
	m.VapidPub, m.VapidPriv = b64(cong.Bytes()), b64(der)
	if err := DatBiMat(m); err != nil {
		return "", nil, err
	}
	return m.VapidPub, k, nil
}

func docKhoaVapid(privB64 string) (*ecdsa.PrivateKey, error) {
	der, err := giaiB64(privB64)
	if err != nil {
		return nil, errors.New("khóa VAPID trong data/bi-mat.yaml không đọc được")
	}
	bk, err := x509.ParsePKCS8PrivateKey(der)
	if err != nil {
		return nil, errors.New("khóa VAPID trong data/bi-mat.yaml không đúng dạng")
	}
	k, ok := bk.(*ecdsa.PrivateKey)
	if !ok {
		return nil, errors.New("khóa VAPID không phải khóa ECDSA")
	}
	return k, nil
}

// jwtVapid ký một JWT ES256 cho đúng một dịch vụ push. "aud" phải là gốc của
// endpoint, không phải cả đường: dịch vụ push đối chiếu đúng chuỗi đó rồi từ
// chối 401 nếu lệch.
func jwtVapid(k *ecdsa.PrivateKey, aud string) (string, error) {
	dau := b64([]byte(`{"typ":"JWT","alg":"ES256"}`))
	// 12 giờ: đủ dài để một lượt gửi nhiều máy dùng chung một token, đủ ngắn
	// để đúng chuẩn (RFC 8292 chặn trên 24 giờ).
	than := map[string]any{
		"aud": aud,
		"exp": time.Now().Add(12 * time.Hour).Unix(),
		"sub": chuVapid(),
	}
	tb, err := json.Marshal(than)
	if err != nil {
		return "", err
	}
	noi := dau + "." + b64(tb)
	bam := sha256.Sum256([]byte(noi))
	r, s, err := ecdsa.Sign(rand.Reader, k, bam[:])
	if err != nil {
		return "", err
	}
	// ES256 muốn r và s ghép thẳng, mỗi số đúng 32 byte. Số nào nhỏ hơn thì
	// đệm 0 ở ĐẦU — cắt ngắn hay đệm ở cuối là chữ ký sai mà vẫn đúng độ dài,
	// dịch vụ push trả 401 không nói vì sao.
	ky := make([]byte, 64)
	demTrai(ky[:32], r)
	demTrai(ky[32:], s)
	return noi + "." + b64(ky), nil
}

func demTrai(ra []byte, n *big.Int) {
	b := n.Bytes()
	if len(b) > len(ra) {
		b = b[len(b)-len(ra):]
	}
	copy(ra[len(ra)-len(b):], b)
}

// chuVapid — trường "sub", để dịch vụ push liên lạc được nếu bot của ta gửi
// sai. Bắt buộc là mailto: hoặc https:.
func chuVapid() string {
	if e := emailChu(); HopLeEmail(e) {
		return "mailto:" + e
	}
	if g := gocWeb(); strings.HasPrefix(g, "https://") {
		return g
	}
	return "mailto:chu@localhost"
}

// --- Mã hóa thân tin (RFC 8291) -----------------------------------------

const (
	// Cỡ bản ghi khai trong tiêu đề. 4096 là mức mọi dịch vụ push đều nhận.
	pushCoBanGhi = 4096
	// Chặn trên cho thân tin. Tin của ta dài vài trăm byte; cắt ở đây chỉ để
	// một sự kiện lạ không làm cả lượt gửi trả 413.
	pushToiDa = 3000
)

// maHoaPush dựng thân request "Content-Encoding: aes128gcm". Khóa mã hóa sinh
// từ ECDH giữa một cặp khóa dùng một lần của ta và khóa công khai của máy
// nhận, nên chính dịch vụ push cũng không đọc được nội dung.
func maHoaPush(uaPub, uaAuth, chu []byte) ([]byte, error) {
	nhan, err := ecdh.P256().NewPublicKey(uaPub)
	if err != nil {
		return nil, fmt.Errorf("khóa p256dh của máy không hợp lệ: %w", err)
	}
	ta, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	chung, err := ta.ECDH(nhan)
	if err != nil {
		return nil, err
	}
	muoi := make([]byte, 16)
	if _, err := rand.Read(muoi); err != nil {
		return nil, err
	}
	taPub := ta.PublicKey().Bytes()

	// Bước riêng của Web Push: trộn thêm auth secret và CẢ HAI khóa công khai
	// vào IKM. Thiếu đoạn này thì tin vẫn mã hóa đúng chuẩn 8188 nhưng trình
	// duyệt giải ra rác rồi bỏ im, không có lỗi nào hiện ra.
	prkKhoa, err := hkdf.Extract(sha256.New, chung, uaAuth)
	if err != nil {
		return nil, err
	}
	tin := append([]byte("WebPush: info\x00"), uaPub...)
	tin = append(tin, taPub...)
	ikm, err := hkdf.Expand(sha256.New, prkKhoa, string(tin), 32)
	if err != nil {
		return nil, err
	}

	prk, err := hkdf.Extract(sha256.New, ikm, muoi)
	if err != nil {
		return nil, err
	}
	cek, err := hkdf.Expand(sha256.New, prk, "Content-Encoding: aes128gcm\x00", 16)
	if err != nil {
		return nil, err
	}
	so, err := hkdf.Expand(sha256.New, prk, "Content-Encoding: nonce\x00", 12)
	if err != nil {
		return nil, err
	}
	kh, err := aes.NewCipher(cek)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(kh)
	if err != nil {
		return nil, err
	}
	// 0x02 đóng bản ghi cuối. Cả tin của ta gói trong đúng một bản ghi, nên
	// byte này luôn là 0x02 chứ không phải 0x01.
	ma := gcm.Seal(nil, so, append(chu, 0x02), nil)

	var b bytes.Buffer
	b.Write(muoi)
	binary.Write(&b, binary.BigEndian, uint32(pushCoBanGhi))
	b.WriteByte(byte(len(taPub)))
	b.Write(taPub)
	b.Write(ma)
	return b.Bytes(), nil
}

// --- Gửi -----------------------------------------------------------------

// errPushChet: dịch vụ push nói endpoint này không còn. Người dùng đã gỡ app
// hay rút quyền — xóa khỏi danh sách, không thử lại.
var errPushChet = errors.New("máy đã ngừng nhận")

// guiPushSuKien — kênh push của BaoTin. Gửi lần lượt từng máy; một máy hỏng
// không chặn máy còn lại.
func guiPushSuKien(sk SuKien) (string, error) {
	ds := DanhSachPushMay()
	if len(ds) == 0 {
		return "chưa máy nào bật thông báo", errKenhTat
	}
	pub, k, err := khoaVapid()
	if err != nil {
		return "", err
	}
	duong := sk.Duong
	if duong == "" {
		duong = "/qt"
	}
	chu, err := json.Marshal(map[string]string{
		"tieu_de": sk.TieuDe,
		"than":    sk.Than,
		"duong":   duong,
		"the":     sk.Loai,
	})
	if err != nil {
		return "", err
	}
	if len(chu) > pushToiDa {
		// Cắt phần thân rồi dựng lại: cắt thẳng chuỗi JSON là JSON hỏng.
		chu, _ = json.Marshal(map[string]string{
			"tieu_de": sk.TieuDe, "than": "", "duong": duong, "the": sk.Loai,
		})
	}

	xong, chet := 0, 0
	var loiCuoi error
	for _, m := range ds {
		err := guiMotMay(m, pub, k, chu)
		switch {
		case err == nil:
			xong++
		case errors.Is(err, errPushChet):
			XoaPushMay(m.Endpoint)
			chet++
		default:
			ghiLoiPushMay(m.Endpoint, err.Error())
			loiCuoi = err
		}
	}
	mo := fmt.Sprintf("%d/%d máy", xong, len(ds))
	if chet > 0 {
		mo += fmt.Sprintf(", bỏ %d máy đã ngừng nhận", chet)
	}
	if xong == 0 && loiCuoi != nil {
		return mo, loiCuoi
	}
	return mo, nil
}

func guiMotMay(m PushMay, pub string, k *ecdsa.PrivateKey, chu []byte) error {
	uaPub, err := giaiB64(m.P256dh)
	if err != nil {
		return err
	}
	uaAuth, err := giaiB64(m.Auth)
	if err != nil {
		return err
	}
	than, err := maHoaPush(uaPub, uaAuth, chu)
	if err != nil {
		return err
	}
	u, err := url.Parse(m.Endpoint)
	if err != nil {
		return err
	}
	jwt, err := jwtVapid(k, u.Scheme+"://"+u.Host)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", m.Endpoint, bytes.NewReader(than))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Content-Encoding", "aes128gcm")
	req.Header.Set("TTL", "86400")
	// Urgency high: đây là việc phải xử trong ngày. Mặc định "normal" cho
	// phép dịch vụ push dồn tin lại chờ máy tỉnh, có khi tới sáng hôm sau.
	req.Header.Set("Urgency", "high")
	req.Header.Set("Authorization", "vapid t="+jwt+", k="+pub)

	resp, err := mailClient.Do(req)
	if err != nil {
		return fmt.Errorf("không gọi được dịch vụ push: %w", err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		return nil
	case resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone:
		return errPushChet
	}
	return fmt.Errorf("dịch vụ push trả %d: %s", resp.StatusCode, catBot(gonMotDong(string(b)), 200))
}

// --- Các đường /qt/push --------------------------------------------------
//
// Chỉ chủ trạm mới bật được: người nhận thông báo đã chốt là một mình Kendy.
// Thợ bật được thì mỗi đơn khách gửi web lại rung cả máy thợ lúc nửa đêm.

// hQtPushKhoa trả khóa công khai VAPID cho JS. Sinh cặp khóa nếu chưa có —
// đây là chỗ duy nhất trong luồng thường gọi tới việc sinh khóa, và nó chỉ
// chạy khi Kendy thật sự bấm nút bật thông báo.
func hQtPushKhoa(w http.ResponseWriter, r *http.Request) {
	pub, err := KhoaVapidCong()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(map[string]string{"khoa": pub})
}

func hQtPushDangKy(w http.ResponseWriter, r *http.Request) {
	var xin struct {
		Endpoint string            `json:"endpoint"`
		Keys     map[string]string `json:"keys"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 8*1024)).Decode(&xin); err != nil {
		http.Error(w, "không đọc được đăng ký", http.StatusBadRequest)
		return
	}
	m := PushMay{
		Endpoint: xin.Endpoint,
		P256dh:   xin.Keys["p256dh"],
		Auth:     xin.Keys["auth"],
		May:      r.UserAgent(),
	}
	if err := ThemPushMay(m); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	nd, _ := NguoiDangNhap(r)
	GhiNhatKy(MucNhatKy{
		Ai: nd.TenHienThi(), IP: ipCua(r), Viec: "push-dang-ky",
		KetQua: fmt.Sprintf("%d máy", len(DanhSachPushMay())),
	})
	w.WriteHeader(http.StatusNoContent)
}

func hQtPushBo(w http.ResponseWriter, r *http.Request) {
	var xin struct {
		Endpoint string `json:"endpoint"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 8*1024)).Decode(&xin); err != nil {
		http.Error(w, "không đọc được yêu cầu", http.StatusBadRequest)
		return
	}
	if err := XoaPushMay(strings.TrimSpace(xin.Endpoint)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
