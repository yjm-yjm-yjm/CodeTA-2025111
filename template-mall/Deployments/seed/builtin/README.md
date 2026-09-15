# 内置模板种子

来源：`D:\金山办公\模板库` 中的 4 个文件（已复制到 `files/`）。

| 名称 | 类型 | 价格 |
|---|---|---|
| 小清新邀请函 | docx / Word | 免费 |
| 工作计划表 | xlsx / Excel | 免费 |
| 中国风康熙五彩蝴蝶纹 | pptx / PPT | ¥1.99 |
| 蚂蚁的秘密 | pdf | ¥0.99 |

封面在 `covers/`（首页预览图）。可用真实首页重新生成封面：

```powershell
python .\seed\builtin\gen_real_covers.py
powershell -File .\scripts\refresh-builtin-covers.ps1
```

完整重新灌库：

```powershell
cd Deployments
powershell -File .\scripts\seed-builtin-templates.ps1
```

后续新模板请走管理端上传（可附带首页预览图；Word/PPT 未传封面时会尝试自动抽取）。
