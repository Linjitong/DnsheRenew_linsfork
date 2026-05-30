# DnsheRenew

`DnsheRenew` 是一个面向 [DNSHE](https://www.dnshe.com/) 免费子域名的自动续期工具。

此项目每次执行时调用 DNSHE 官方 API，自动找出已经进入续期窗口的子域名并完成续期，避免手动登录网站逐个处理。适合放在 GitHub Actions、服务器定时任务或本地 cron 中长期运行。
此项目会解析域名到期时间，只对剩余时间小于 `180` 天的子域名发起续期请求，从而减少无效请求。

## 适用场景

- 你有一个或多个 DNSHE 免费子域名，需要长期自动续期
- 你有多组 API 凭证，希望统一管理和批量执行
- 你希望公开日志尽量干净，但私有通知里能看到完整的域名和到期信息
- 你希望把任务挂到 GitHub Actions 或其他 CI / 定时任务平台上

## 功能特性

- 自动识别进入续期窗口的域名，只续期剩余时间小于 `180` 天的子域名
- 支持多账号批量执行，单个账号失败不会中断其他账号
- 支持 `dry-run` 演练模式，先看匹配结果再决定是否真实执行
- 遇到 DNSHE 限流时会按 `reset_at` 做有限次数退避重试
- 支持控制台调试输出、Telegram 机器人通知、Lark 通知、Webhook 通知
- 私有通知会带每个账号的域名列表、当前到期时间、续期结果和失败原因
- 公开输出默认只保留 `renewed_total` 和脱敏后的错误摘要
- 内置 DNSHE SDK，可单独复用 DNSHE API 能力

## 工作方式

程序每次执行时会按下面的流程运行：

1. 读取环境变量并加载一个或多个 DNSHE API 凭证
2. 拉取每个账号下的子域名列表
3. 解析剩余时间和到期时间，只保留进入续期窗口的域名
4. 对命中的域名发起续期请求
5. 汇总结果，并按配置发送私有通知

如果某个账号下没有任何域名进入续期窗口，会按空操作成功处理，不会作为失败退出。

## GitHub Actions

这是此项目最推荐的运行方式。

仓库根目录已经提供 `composite action`，你可以在自己的 GitHub 仓库里直接调用。
完整的 action 输入、输出和更多示例见 [github-action.md](docs/github-action.md)。
如果你想直接复制一份完整 workflow，可以使用带注释的示例文件 [dnshe-renew.yml](examples/dnshe-renew.yml)。

推荐使用方式：

1. 在 GitHub 上创建一个你自己的仓库，用来放 workflow
2. 复制 [dnshe-renew.yml](examples/dnshe-renew.yml) 到你自己仓库的 `.github/workflows/dnshe-renew.yml`
3. 按需取消注释并修改里面的可选项，最小可用版本如下

```yaml
name: Monthly DNSHE Renew

on:
  schedule:
    - cron: "15 0 1 * *"
  workflow_dispatch:

jobs:
  renew:
    runs-on: ubuntu-latest
    environment:
      name: dnshe
    steps:
      - name: Renew DNSHE domains
        id: renew
        uses: nhirsama/DnsheRenew@v0.1
        with:
          api-keys: ${{ secrets.DNSHE_API_KEYS }}
          api-secrets: ${{ secrets.DNSHE_API_SECRETS }}
```

4. 在你自己的仓库里配置 `Secrets` / `Variables`
   常见至少包括：
   - `DNSHE_API_KEYS`
   - `DNSHE_API_SECRETS`
   - 如需通知，再按 [github-action.md](docs/github-action.md) 增加 Telegram、Lark、Webhook 相关配置
   - 如果你使用的是 `Environment secrets`，请把环境名统一设置为 `dnshe`，并在 workflow 的 job 上声明 `environment: dnshe`
5. 提交 workflow 文件并确保仓库已启用 Actions
6. 需要立即测试时，可以在 Actions 页面手动触发 `workflow_dispatch`

如果你不想自己从零建 workflow，也可以直接使用仓库右上角的 `Use this template` 创建你自己的仓库，再按文档补配置。

如果你打算长期依赖 GitHub Actions 的计划任务，建议注意下面几点：

- 公共仓库在 `60` 天无仓库活动时，GitHub 会自动禁用 scheduled workflows
- 私有仓库不受这个公开仓库限制影响
- 如果你希望更稳定地长期运行，建议将模板仓库创建为私有仓库

## 环境变量

必填：

- `DNSHE_API_KEYS`
- `DNSHE_API_SECRETS`

可选：

- `DNSHE_DRY_RUN`
- `DNSHE_API_BASE_URL`
- `DNSHE_RUN_TIMEOUT`
- `DNSHE_DEBUG`
- `DNSHE_NOTIFY_TELEGRAM_BOT_TOKEN`
- `DNSHE_NOTIFY_TELEGRAM_CHAT_ID`
- `DNSHE_NOTIFY_TELEGRAM_MESSAGE_THREAD_ID`
- `DNSHE_NOTIFY_WEBHOOK_URL`
- `DNSHE_NOTIFY_WEBHOOK_TOKEN`

凭证规则：

- 始终使用 `DNSHE_API_KEYS` 和 `DNSHE_API_SECRETS` 这两个复数环境变量
- 单账号场景也按列表处理，只填 1 组即可
- 两个列表的项目数量必须一致
- 分隔符支持 `,`、`;` 和换行

详细的通知模块环境变量说明见 [notification.md](docs/notification.md)。

## 通知说明

当前内建四种通知器：

- `Console`
  仅在 `DNSHE_DEBUG=true` 时启用
- `Telegram`
  需要同时配置 `DNSHE_NOTIFY_TELEGRAM_BOT_TOKEN` 和 `DNSHE_NOTIFY_TELEGRAM_CHAT_ID`
- `Lark`
  支持 webhook 和 custom app 两种模式
- `Webhook`
  需要配置 `DNSHE_NOTIFY_WEBHOOK_URL`

私有通知内容会包含：

- 每个账号的匹配数量、续期数量、失败数量
- 每个账号下的域名列表、当前到期时间、剩余天数
- 续期成功域名的新到期时间
- 失败域名及失败原因
- 脱敏后的 API Key 标识

Telegram 通知会使用格式化消息输出，并在内容较长时自动分段。

当前工作流使用的配置来源如下：

- GitHub Secrets
  - `DNSHE_API_KEYS`
  - `DNSHE_API_SECRETS`
  - `DNSHE_NOTIFY_TELEGRAM_BOT_TOKEN`
  - `DNSHE_NOTIFY_TELEGRAM_CHAT_ID`
  - `DNSHE_NOTIFY_TELEGRAM_MESSAGE_THREAD_ID`
  - `DNSHE_NOTIFY_LARK_WEBHOOK_URL`
  - `DNSHE_NOTIFY_WEBHOOK_URL`
  - `DNSHE_NOTIFY_WEBHOOK_TOKEN`
- GitHub Variables
  - `DNSHE_API_BASE_URL`
  - `DNSHE_RUN_TIMEOUT`

## 本地运行

最小运行方式：

```bash
DNSHE_API_KEYS="cfsd_xxx" \
DNSHE_API_SECRETS="yyy" \
go run ./cmd/dnsherene
```

多账号：

```bash
DNSHE_API_KEYS="key_1,key_2" \
DNSHE_API_SECRETS="secret_1,secret_2" \
go run ./cmd/dnsherene
```

演练模式：

```bash
DNSHE_API_KEYS="cfsd_xxx" \
DNSHE_API_SECRETS="yyy" \
DNSHE_DRY_RUN=true \
go run ./cmd/dnsherene
```

调试模式：

```bash
DNSHE_API_KEYS="cfsd_xxx" \
DNSHE_API_SECRETS="yyy" \
DNSHE_DEBUG=true \
go run ./cmd/dnsherene
```

开启 `DNSHE_DEBUG=true` 后，详细通知会同步输出到控制台，便于本地排查。

整体超时：

```bash
DNSHE_API_KEYS="cfsd_xxx" \
DNSHE_API_SECRETS="yyy" \
DNSHE_RUN_TIMEOUT=10m \
go run ./cmd/dnsherene
```

`DNSHE_RUN_TIMEOUT` 使用 Go duration 格式，例如 `5m`、`10m`、`30s`；默认 `10m`。

## Docker / Compose

项目支持以容器方式部署，并在容器内使用 cron 定时执行。

最小使用方式：

1. 复制 [compose.example.yml](compose.example.yml) 为你自己的 `compose.yml`
2. 填入 `DNSHE_API_KEYS`、`DNSHE_API_SECRETS`
3. 按需修改 `DNSHE_CRON_SCHEDULE` 和 `TZ`
4. 执行：

```bash
docker compose up -d --build
```

默认示例中的 cron 表达式为每月 `1` 号 `00:15` 执行一次，时区为 `Asia/Shanghai`。

如果你只想立即手动执行一次，而不是等待下一个 cron 时间点：

```bash
docker compose run --rm dnsherene run
```

## 隐私与日志

- 公开日志默认不会打印域名、到期时间、剩余天数、Webhook 地址或原始 API Key
- 成功时公开输出只有 `renewed_total=<number>`
- 失败时会输出脱敏后的错误摘要
- 私有通知和调试日志会包含详细域名信息，因此更适合发往 Telegram、Webhook 或本地控制台

## 项目结构

项目按“入口、执行、通知、SDK”拆分：

- `cmd/dnsherene`
  程序入口，只负责加载配置、执行和输出公开结果
- `internal/runner`
  多账号执行编排
- `internal/notification`
  控制台、Telegram、Webhook 等通知实现与通知请求模型
- `internal/app`
  单账号续期编排逻辑
- `internal/output`
  公开输出与脱敏
- `internal/report`
  执行结果和通知共用的结构化报告模型
- `pkg/dnshe`
  DNSHE API SDK

## DNSHE SDK 覆盖范围

`pkg/dnshe` 当前已覆盖文档中的全部接口分组：

- `subdomains`
  - `ListSubdomains`
  - `RegisterSubdomain`
  - `GetSubdomain`
  - `DeleteSubdomain`
  - `RenewSubdomain`
- `dns_records`
  - `ListDNSRecords`
  - `CreateDNSRecord`
  - `UpdateDNSRecord`
  - `DeleteDNSRecord`
- `keys`
  - `ListAPIKeys`
  - `CreateAPIKey`
  - `DeleteAPIKey`
  - `RegenerateAPIKey`
- `quota`
  - `GetQuota`
- `whois`
  - `NewPublicClient`
  - `Whois`
- `permanent_upgrade`
  - `ListPermanentUpgrades`
  - `CreatePermanentUpgrade`
  - `AssistPermanentUpgrade`
  - `CancelPermanentUpgrade`

SDK 额外处理了这些细节：

- HTTP 错误和 `success=false` 业务错误统一返回 `*dnshe.APIError`
- V2 统一错误结构中的 `error_code`、`details`、`request_id`、限流 `limit/remaining/reset_at` 会保留在结构化错误中
- 子域名列表支持 V2 新增的分页、筛选、排序和字段裁剪参数，并可读取 `count` 与 `pagination`
- DNS 记录支持 V2 扩展字段和类型，包括线路、代理状态、SRV 的 weight/port/target 以及 CAA 的 flag/tag/value
- `RenewSubdomain` 已包含续期相关返回字段
- DNS 记录创建前会做基础参数校验

## 许可
本项目根据 Apache License 2.0 授权 - 详细内容请参阅[LICENSE](./LICENSE)文件。  
版权所有 © 2026 [nhir](https://github.com/nhirsama)。
