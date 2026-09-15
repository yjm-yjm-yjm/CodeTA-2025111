# 模板商城（template-mall）

全栈考核：模板商城微服务。根目录名称固定为 `themproject/template-mall`。

## 1. 工程红线

- 每个代码文件不超过 **1000 行**（含生成代码；proto 需拆分保证生成物合规）。
- 保证健壮性与可运维性：超时、幂等、错误日志、配置外置、优雅退出。
- 六个顶层项目独立启动，禁止跨项目引用内部代码。

## 2. 端口与认证（已确认）

| 端 | 约定 |
|---|---|
| B 端前端 `TemplateAdminFrontend` | **3001** |
| B 端认证 | 独立登录页 + 公司统一 Auth（WPS OAuth）；正式环境登录即管理员 |
| C 端认证 | 自行注册登录 + JWT（不做统一登录） |

其余服务固定端口以作业文档为准；当前 OrderServer 默认 gRPC `:50051`（可配置）。

## 3. 项目组成

| 目录 | 状态 | 说明 |
|---|---|---|
| TemplateOrderServer | 第 1 步已完成 | 核心 gRPC + DB + Kafka 消费 |
| PayWebServer | 第 2 步已完成 | 模拟支付 HTTP |
| TemplateWebServer | 第 3 步已完成 | C 端 BFF + JWT |
| TemplateAdminWebServer | 第 4 步已完成 | B 端 BFF + WPS OAuth + OSS 直传 |
| TemplateFrontend | 第 5 步已完成 | C 端前端（:3000） |
| TemplateAdminFrontend | 第 6 步已完成 | B 端前端（:3001） |
| Deployments | 已完成 | docker-compose + e2e 脚本 |

## 4. 提交节奏

每完成一个独立项目后本地提交存档一次（由同学自行 commit）。
