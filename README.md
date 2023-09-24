# Hackergame 邮件网关参考实现

一个简单的 Hackergame 邮件网关参考实现。使用 SMTP 发送邮件，已在中国科学技术大学邮件系统测试。

API Spec 和预期的调用方式参见[此处](https://github.com/ustclug/hackergame/blob/b3a63e67270dc7b1b11491d8692d30caa54d847f/frontend/auth_providers/external.py#L24C5-L24C5)

本参考实现将启动一个 HTTP 服务器，在 `/mail` 接受 POST 请求，并使用 SMTP 将邮件发送至对应邮箱。

⚠️ 本参考实现使用 TLS 连接到 SMTP 服务器。

## 部署

1. 设置以下环境变量:
    - `HG_AUTH_TOKEN`: 用于验证调用方的 Token/Key
    - `HG_SMTP_HOST`: SMTP 主机名
    - `HG_SMTP_PORT`: SMTP 端口
    - `HG_SMTP_USERNAME`: SMTP 用户名
    - `HG_SMTP_PASSWORD`: SMTP 密码
    - `HG_SMTP_IDENTITY`: SMTP 身份，用于 AUTH 和 MAIL FROM 命令
    - `HG_LISTEN_ADDR`: HTTP 监听地址

2. 执行 `go build` 构建或使用 `go run` 直接运行。