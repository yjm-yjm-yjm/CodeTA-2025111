# -*- coding: utf-8 -*-
from PIL import Image, ImageDraw, ImageFont
import os

out_dir = r"D:\sample\yangjiamin\themproject\template-mall\Deployments\seed\builtin\covers"
items = [
    ("xiaqingxin-yaoqinghan.png", "小清新邀请函", "WORD", "#2F6FED", "#EAF2FF"),
    ("gongzuo-jihuabiao.png", "工作计划表", "EXCEL", "#1F7A4C", "#E8F7EF"),
    ("kangxi-hudiewen.png", "中国风康熙五彩蝴蝶纹", "PPT", "#C23B22", "#FDEBE6"),
    ("mayi-demimi.png", "蚂蚁的秘密", "PDF", "#6B3FA0", "#F3ECFA"),
]

W, H = 420, 560
try:
    font_title = ImageFont.truetype("msyh.ttc", 28)
    font_badge = ImageFont.truetype("msyh.ttc", 18)
    font_small = ImageFont.truetype("msyh.ttc", 16)
except Exception:
    font_title = ImageFont.load_default()
    font_badge = font_title
    font_small = font_title

for fname, title, badge, accent, soft in items:
    img = Image.new("RGB", (W, H), "#F4F5F7")
    d = ImageDraw.Draw(img)
    d.rounded_rectangle((48, 36, W-36, H-48), radius=10, fill="#D0D5DD")
    d.rounded_rectangle((40, 28, W-48, H-56), radius=10, fill="white", outline="#E4E7EC")
    d.rectangle((40, 28, W-48, 96), fill=accent)
    d.text((56, 48), badge, fill="white", font=font_badge)
    y = 120
    for i, w in enumerate([0.85, 0.72, 0.9, 0.55, 0.78, 0.66, 0.8, 0.5]):
        d.rounded_rectangle((64, y, 64+int((W-140)*w), y+14), radius=4, fill=soft if i % 2 == 0 else "#EEF1F5")
        y += 28
    d.rectangle((40, H-140, W-48, H-56), fill="#FAFBFC")
    tw = W - 120
    lines, cur = [], ""
    for ch in title:
        t2 = cur + ch
        if d.textlength(t2, font=font_title) <= tw:
            cur = t2
        else:
            lines.append(cur); cur = ch
    if cur: lines.append(cur)
    ty = H - 125
    for line in lines[:2]:
        d.text((56, ty), line, fill="#1D2939", font=font_title)
        ty += 34
    d.text((56, H-78), "模板商城 · 首屏预览", fill="#98A2B3", font=font_small)
    path = os.path.join(out_dir, fname)
    img.save(path, "PNG")
    print("cover", path)
print("done")
