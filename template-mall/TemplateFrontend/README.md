# TemplateFrontend

C 端用户前端（React + TypeScript + Vite）。

## 1. 功能

- 注册 / 登录（JWT）
- 已上架模板列表与下载
- 需支付时跳转模拟支付页
- 我的订单 / 取消未支付零售订单

## 2. 启动

```bash
cp .env.example .env
npm install
npm run dev
```

默认端口：**3000**。API 默认指向 `http://127.0.0.1:8080`（`TemplateWebServer`）。

## 3. 环境变量

见 `.env.example`：

- `VITE_API_BASE`：C 端 BFF 地址

## 4. 与其他服务关系

```text
TemplateFrontend (:3000)
    → HTTP → TemplateWebServer (:8080)
                 → gRPC → TemplateOrderServer
```

## 5. 构建

```bash
npm run build
```
