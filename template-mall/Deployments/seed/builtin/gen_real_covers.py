# -*- coding: utf-8 -*-
"""从模板源文件生成真实首页预览图到 covers/。"""
from __future__ import annotations

import io
import zipfile
import xml.etree.ElementTree as ET
from pathlib import Path

from PIL import Image, ImageDraw, ImageFont

ROOT = Path(__file__).resolve().parent
FILES = ROOT / "files"
COVERS = ROOT / "covers"
COVERS.mkdir(parents=True, exist_ok=True)

W, H = 420, 560


def font(size: int):
    for name in ("msyh.ttc", "msyh.ttf", "arial.ttf"):
        try:
            return ImageFont.truetype(name, size)
        except OSError:
            continue
    return ImageFont.load_default()


def fit_cover(img: Image.Image) -> Image.Image:
    img = img.convert("RGB")
    src_w, src_h = img.size
    scale = max(W / src_w, H / src_h)
    nw, nh = max(1, int(src_w * scale)), max(1, int(src_h * scale))
    img = img.resize((nw, nh), Image.Resampling.LANCZOS)
    left = (nw - W) // 2
    top = (nh - H) // 2
    return img.crop((left, top, left + W, top + H))


def save(img: Image.Image, name: str):
    out = COVERS / name
    fit_cover(img).save(out, "PNG", optimize=True)
    print("cover", out)


def fallback(title: str, badge: str, accent: str, soft: str, out_name: str):
    img = Image.new("RGB", (W, H), "#F4F5F7")
    d = ImageDraw.Draw(img)
    d.rounded_rectangle((40, 28, W - 48, H - 56), radius=10, fill="white", outline="#E4E7EC")
    d.rectangle((40, 28, W - 48, 96), fill=accent)
    d.text((56, 48), badge, fill="white", font=font(18))
    y = 120
    for i, ratio in enumerate([0.85, 0.72, 0.9, 0.55, 0.78, 0.66, 0.8, 0.5]):
        d.rounded_rectangle((64, y, 64 + int((W - 140) * ratio), y + 14), radius=4, fill=soft if i % 2 == 0 else "#EEF1F5")
        y += 28
    d.text((56, H - 120), title[:16], fill="#1D2939", font=font(26))
    d.text((56, H - 78), "模板商城 · 首页预览", fill="#98A2B3", font=font(16))
    save(img, out_name)


def cover_pdf(path: Path, out_name: str):
    import fitz

    doc = fitz.open(path)
    page = doc.load_page(0)
    pix = page.get_pixmap(matrix=fitz.Matrix(2, 2), alpha=False)
    img = Image.open(io.BytesIO(pix.tobytes("png")))
    doc.close()
    save(img, out_name)


def _zip_image(z: zipfile.ZipFile, inner: str) -> Image.Image | None:
    try:
        data = z.read(inner)
    except KeyError:
        return None
    try:
        return Image.open(io.BytesIO(data)).convert("RGB")
    except Exception:
        return None


def cover_pptx(path: Path, out_name: str, title: str):
    with zipfile.ZipFile(path) as z:
        rels_name = "ppt/slides/_rels/slide1.xml.rels"
        slide_name = "ppt/slides/slide1.xml"
        media = None
        if rels_name in z.namelist() and slide_name in z.namelist():
            rels = {}
            root = ET.fromstring(z.read(rels_name))
            for rel in root:
                rid = rel.attrib.get("Id")
                target = rel.attrib.get("Target", "")
                if rid and target:
                    if not target.startswith("ppt/"):
                        target = "ppt/slides/" + target
                        target = str(Path(target).as_posix())
                        # normalize ../media/xxx
                        while "/../" in target:
                            parts = target.split("/")
                            target = "/".join(parts[: parts.index("..") - 1] + parts[parts.index("..") + 1 :])
                        if target.startswith("../"):
                            target = "ppt/" + target[3:]
                    rels[rid] = target.replace("\\", "/")
            slide = ET.fromstring(z.read(slide_name))
            ns = {
                "a": "http://schemas.openxmlformats.org/drawingml/2006/main",
                "r": "http://schemas.openxmlformats.org/officeDocument/2006/relationships",
            }
            embeds = []
            for blip in slide.findall(".//a:blip", ns):
                rid = blip.attrib.get("{http://schemas.openxmlformats.org/officeDocument/2006/relationships}embed")
                if rid and rid in rels:
                    embeds.append(rels[rid])
            for target in embeds:
                media = _zip_image(z, target)
                if media:
                    break
        if media is None:
            # 退化为最大媒体图
            candidates = [n for n in z.namelist() if n.startswith("ppt/media/") and n.lower().endswith((".png", ".jpg", ".jpeg", ".webp"))]
            best = None
            best_size = 0
            for n in candidates:
                info = z.getinfo(n)
                if info.file_size > best_size:
                    best_size = info.file_size
                    best = n
            if best:
                media = _zip_image(z, best)
        if media is None:
            fallback(title, "PPT", "#C23B22", "#FDEBE6", out_name)
            return
        save(media, out_name)


def cover_docx(path: Path, out_name: str, title: str):
    with zipfile.ZipFile(path) as z:
        candidates = [n for n in z.namelist() if n.startswith("word/media/") and n.lower().endswith((".png", ".jpg", ".jpeg", ".webp"))]
        # 优先 image1，否则取最大图
        preferred = [n for n in candidates if Path(n).name.lower().startswith("image1.")]
        chosen = preferred[0] if preferred else None
        if not chosen and candidates:
            chosen = max(candidates, key=lambda n: z.getinfo(n).file_size)
        if not chosen:
            fallback(title, "WORD", "#2F6FED", "#EAF2FF", out_name)
            return
        img = _zip_image(z, chosen)
        if img is None:
            fallback(title, "WORD", "#2F6FED", "#EAF2FF", out_name)
            return
        save(img, out_name)


def cover_xlsx(path: Path, out_name: str, title: str):
    import openpyxl

    wb = openpyxl.load_workbook(path, read_only=True, data_only=True)
    ws = wb.active
    rows = []
    for i, row in enumerate(ws.iter_rows(max_row=12, max_col=6, values_only=True)):
        rows.append([("" if v is None else str(v))[:18] for v in row])
        if i >= 11:
            break
    wb.close()

    img = Image.new("RGB", (W, H), "#F7FBF8")
    d = ImageDraw.Draw(img)
    d.rectangle((0, 0, W, 64), fill="#1F7A4C")
    d.text((20, 18), title[:14], fill="white", font=font(22))
    d.text((20, 72), "Excel · 首页预览", fill="#667788", font=font(14))
    y = 100
    f = font(13)
    for r_i, row in enumerate(rows):
        x = 16
        bg = "#FFFFFF" if r_i % 2 == 0 else "#E8F7EF"
        d.rectangle((12, y - 4, W - 12, y + 28), fill=bg)
        for cell in row:
            d.text((x, y), cell, fill="#1D2939", font=f)
            x += 66
        y += 34
        if y > H - 40:
            break
    save(img, out_name)


def main():
    jobs = [
        ("xiaqingxin-yaoqinghan.docx", "xiaqingxin-yaoqinghan.png", "docx", "小清新邀请函"),
        ("gongzuo-jihuabiao.xlsx", "gongzuo-jihuabiao.png", "xlsx", "工作计划表"),
        ("kangxi-hudiewen.pptx", "kangxi-hudiewen.png", "pptx", "中国风康熙五彩蝴蝶纹"),
        ("mayi-demimi.pdf", "mayi-demimi.png", "pdf", "蚂蚁的秘密"),
    ]
    for src, out, kind, title in jobs:
        path = FILES / src
        print("render", src)
        if kind == "pdf":
            cover_pdf(path, out)
        elif kind == "pptx":
            cover_pptx(path, out, title)
        elif kind == "docx":
            cover_docx(path, out, title)
        elif kind == "xlsx":
            cover_xlsx(path, out, title)
    print("done")


if __name__ == "__main__":
    main()
