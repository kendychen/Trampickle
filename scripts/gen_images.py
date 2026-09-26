from PIL import Image, ImageDraw, ImageFont
import pathlib
import fitz  # PyMuPDF - render SVG that cairosvg/svglib need cairo

OUT = pathlib.Path(r"D:\Dự Án Sửa Chữa Pickleball\content\hinh-anh")
OUT.mkdir(parents=True, exist_ok=True)
OUT2 = pathlib.Path(r"D:\Dự Án Sửa Chữa Pickleball\giao-trinh-sua-vot\anh")
OUT2.mkdir(parents=True, exist_ok=True)
OUT3 = pathlib.Path(r"D:\Dự Án Sửa Chữa Pickleball\dist\giao-trinh-sua-vot\anh")
OUT3.mkdir(parents=True, exist_ok=True)

W, H = 1080, 1080
BG = (255,255,255)
DARK = (11,61,46)
YELLOW = (244,180,0)
GRAY = (100,100,100)

# --- T-Pic logo: render truc tiep tu SVG goc bang PyMuPDF (khong can cairo) ---
SRC_TOP = pathlib.Path(r"D:\TramVot\assets\logo\T-Pic\nen-sang.svg")  # tim+vang tren nen sang -> header trang
SRC_FOOT = pathlib.Path(r"D:\TramVot\assets\logo\T-Pic\nen-toi.svg")  # giay+vang tren nen toi -> footer xanh dam

def svg_to_pil(svg_path: pathlib.Path, target_w: int):
    doc = fitz.open(str(svg_path))
    page = doc[0]
    scale = target_w / page.rect.width
    pix = page.get_pixmap(matrix=fitz.Matrix(scale, scale), alpha=True)
    im = Image.frombytes("RGBA" if pix.alpha else "RGB", [pix.width, pix.height], pix.samples)
    if im.mode != "RGBA":
        im = im.convert("RGBA")
    doc.close()
    return im

LOGO_TOP = svg_to_pil(SRC_TOP, 220)
LOGO_FOOT = svg_to_pil(SRC_FOOT, 180)
print(f"logo TOP {LOGO_TOP.size} from {SRC_TOP.name}, FOOT {LOGO_FOOT.size} from {SRC_FOOT.name} (fitz)")

def get_font(size, bold=False):
    candidates = [
        r"C:\Windows\Fonts\tahomabd.ttf" if bold else r"C:\Windows\Fonts\tahoma.ttf",
        r"C:\Windows\Fonts\segoeui.ttf",
    ]
    for p in candidates:
        try:
            return ImageFont.truetype(p, size)
        except:
            pass
    return ImageFont.load_default()

items = [
    ("01-bang-gia-2025.jpg", "BẢNG GIÁ\nSỬA VỢT 2025", "Nứt viền 250-450K  •  Tách lớp 300-550K\nĐiểm chết 200-400K  •  Thay grip 80-150K", "TRẠM PICKLE  •  58 TỐ HỮU - HÀ NỘI"),
    ("02-nut-vien.jpg", "VỢT NỨT VIỀN\nCÓ SỬA ĐƯỢC KHÔNG?", "2-5cm: sửa đẹp 90% - NÊN SỬA\n5-10cm: gia cường carbon\n>10cm: nên mua mới", "TRẠM PICKLE"),
    ("03-tach-lop.jpg", "MẶT VỢT\nPHỒNG RỘP", "TÁCH LỚP - ÉP LẠI 300-550K\nẤn lún • Gõ kêu bộp • Bóng tịt", "TRẠM PICKLE"),
    ("04-diem-chet.jpg", "ĐIỂM CHẾT\nVỢT PICKLEBALL", "TEST 30 GIÂY: GÕ ĐỀU MẶT VỢT\nChỗ kêu tịt là dính - Sửa 200-400K", "TRẠM PICKLE"),
    ("05-de-giay-mon.jpg", "ĐẾ GIÀY\nMÒN NHẴN", "TRƯỢT NGÃ + ĐAU GỐI\n5 dấu hiệu phải thay ngay!", "TRẠM PICKLE"),
    ("06-ve-sinh-giay.jpg", "VỆ SINH GIÀY\nĐÚNG CÁCH", "Không giặt máy • Không phơi nắng gắt\nGiặt tay - Phơi gió - Baking soda", "TRẠM PICKLE"),
    ("07-thay-grip.jpg", "THAY GRIP\n80-150K", "10 PHÚT LẤY NGAY\n2-4 tháng thay 1 lần", "TRẠM PICKLE  •  58 TỐ HỮU"),
    ("08-bao-quan.jpg", "7 THÓI QUEN\nVỢT BỀN GẤP ĐÔI", "Không để cốp nắng • Dán viền\nBao vợt • Lau khô • Gõ kiểm tra", "TRẠM PICKLE"),
]

for fname, title, subtitle, footer in items:
    img = Image.new("RGB", (W, H), BG)
    d = ImageDraw.Draw(img)
    d.rectangle([0, 0, W, 18], fill=DARK)

    # paste T-Pic logo: nen-sang (tim+vang) goc tren phai, nen trong suot
    if LOGO_TOP is not None:
        img.paste(LOGO_TOP, (W - LOGO_TOP.width - 28, 30), LOGO_TOP)

    font_title = get_font(78, bold=True)
    font_sub = get_font(30, bold=False)
    font_footer = get_font(26, bold=True)

    lines = title.split("\n")
    y = 170
    for line in lines:
        bbox = d.textbbox((0, 0), line, font=font_title)
        w = bbox[2] - bbox[0]
        d.text(((W - w) // 2, y), line, fill=DARK, font=font_title)
        y += 92
    y += 18
    d.rectangle([(W - 200) // 2, y, (W + 200) // 2, y + 8], fill=YELLOW)
    y += 40
    for line in subtitle.split("\n"):
        bbox = d.textbbox((0, 0), line, font=font_sub)
        w = bbox[2] - bbox[0]
        d.text(((W - w) // 2, y), line, fill=(60, 60, 60), font=font_sub)
        y += 44

    # footer bar + T-Pic nen-toi (giay+vang) ben trai
    d.rectangle([0, H - 90, W, H], fill=DARK)
    if LOGO_FOOT is not None:
        # can giua doc trong bar 90px: bar top H-90, logo h ~42
        y_foot = H - 90 + (90 - LOGO_FOOT.height) // 2
        img.paste(LOGO_FOOT, (24, y_foot), LOGO_FOOT)
    bbox = d.textbbox((0, 0), footer, font=font_footer)
    w = bbox[2] - bbox[0]
    d.text(((W - w) // 2, H - 62), footer, fill=(255, 255, 255), font=font_footer)

    small_font = get_font(20)
    d.text((W - 210, H - 125), "trampickle.vn", fill=GRAY, font=small_font)

    for outdir in [OUT, OUT2, OUT3]:
        img.save(outdir / fname, "JPEG", quality=88, optimize=True)
    print("created", fname)

og_src = Image.open(OUT / "01-bang-gia-2025.jpg")
og = og_src.resize((1200, 1200), Image.LANCZOS).crop((0, 60, 1200, 690))
for outdir in [OUT, OUT2, OUT3]:
    og.save(outdir / "og-bang-gia-2025.jpg", "JPEG", quality=85)
print("created og-bang-gia-2025.jpg")
print("DONE")

