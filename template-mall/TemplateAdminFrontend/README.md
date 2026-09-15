# TemplateAdminFrontend

B 端管理前端（React + TypeScript + Vite），端口固定 **3001**。

## 1. 功能

- 管理员登录（WPS OAuth）
- 模板鉴权直传上传、创建、改价、上下架
- 订单列表
- 设置 / 取消用户会员

## 2. 启动

```bash
cp .env.example .env
npm install
npm run dev
```

默认：`http://127.0.0.1:3001` → API `http://127.0.0.1:8081`

## 3. 环境变量

- `VITE_API_BASE`：Admin BFF 地址

## 4. 调用关系

```text
TemplateAdminFrontend (:3001)
    → HTTP → TemplateAdminWebServer (:8081)
                 → gRPC → TemplateOrderServer
                 → OSS 预签名直传
```

WPS 回调成功后会跳回本前端并带上 `?token=`，页面会自动写入本地会话。

## 5. 构建

```bash
npm run build
```
