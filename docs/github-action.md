# GitHub Action 使用说明

`dnsherene` 仓库根目录提供了一个 `composite action`，适合在其他仓库的 workflow 里直接调用。

如果你想直接复制一份完整 workflow，可以使用带注释的示例文件 [dnshe-renew.yml](../examples/dnshe-renew.yml)。该文件放在 `examples/` 目录下。

## 基本用法

最小示例：

```yaml
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

      - name: Print result
        run: echo "renewed_total=${{ steps.renew.outputs.renewed-total }}"
```

带通知配置的示例：

```yaml
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
          api-base-url: ${{ vars.DNSHE_API_BASE_URL }}
          run-timeout: ${{ vars.DNSHE_RUN_TIMEOUT }}
          telegram-bot-token: ${{ secrets.DNSHE_NOTIFY_TELEGRAM_BOT_TOKEN }}
          telegram-chat-id: ${{ secrets.DNSHE_NOTIFY_TELEGRAM_CHAT_ID }}
          telegram-message-thread-id: ${{ secrets.DNSHE_NOTIFY_TELEGRAM_MESSAGE_THREAD_ID }}
          lark-webhook-url: ${{ secrets.DNSHE_NOTIFY_LARK_WEBHOOK_URL }}
          webhook-url: ${{ secrets.DNSHE_NOTIFY_WEBHOOK_URL }}
          webhook-token: ${{ secrets.DNSHE_NOTIFY_WEBHOOK_TOKEN }}
```

## Inputs

| 输入名 | 说明 | 是否必须 | 示例 |
| --- | --- | --- | --- |
| `api-keys` | DNSHE API Key 列表，支持逗号、分号或换行分隔 | 是 | `key_a,key_b` |
| `api-secrets` | DNSHE API Secret 列表，顺序需与 `api-keys` 一一对应 | 是 | `secret_a,secret_b` |
| `api-base-url` | DNSHE API 基础地址 | 否 | `https://dnshe.example.com` |
| `dry-run` | 是否启用演练模式 | 否 | `true` |
| `run-timeout` | 单次执行整体超时，使用 Go duration 格式 | 否 | `10m` |
| `debug` | 是否启用控制台调试通知 | 否 | `true` |
| `telegram-bot-token` | Telegram Bot Token | 否 | `123456:abcdef` |
| `telegram-chat-id` | Telegram 目标聊天 ID | 否 | `-1001234567890` |
| `telegram-message-thread-id` | Telegram topic / thread ID | 否 | `7` |
| `telegram-parse-mode` | Telegram 解析模式 | 否 | `HTML` |
| `lark-webhook-url` | 飞书群机器人 webhook 地址 | 否 | `https://open.feishu.cn/open-apis/bot/v2/hook/xxx` |
| `lark-app-id` | 飞书自建应用 App ID | 否 | `cli_xxx` |
| `lark-app-secret` | 飞书自建应用 App Secret | 否 | `yyy` |
| `lark-receiver-type` | 飞书接收人 ID 类型 | 否 | `chat_id` |
| `lark-receivers` | 飞书接收人列表，支持逗号、分号或换行分隔 | 否 | `oc_xxx,oc_yyy` |
| `webhook-url` | 自定义 webhook 地址 | 否 | `https://example.com/hooks/dnshe` |
| `webhook-token` | 自定义 webhook Bearer Token | 否 | `token-123` |

## Outputs

| 输出名 | 说明 | 示例 |
| --- | --- | --- |
| `renewed-total` | 本次续期成功的域名总数 | `3` |
| `notification-error-count` | 本次通知发送失败数量 | `1` |
| `error-count` | 本次主流程错误数量 | `0` |
| `success` | 本次执行是否成功完成 | `true` |

## 说明

- 该 action 内部会把 `with:` 传入的参数映射为程序现有的 `DNSHE_*` 环境变量，再执行 CLI。
- 如果主流程执行失败，action 会返回失败状态。
- 如果只是通知发送失败，action 仍会继续返回主流程结果，同时在日志里输出 `notification_error_*`。
- GitHub Actions 的敏感信息建议放在 `Secrets`，例如 `api-keys`、`api-secrets`、机器人 token、webhook 地址。
- 如果你使用的是 `Environment secrets`，请把环境名统一设置为 `dnshe`，并在 job 上声明 `environment: dnshe`；否则 workflow 运行时拿不到这些 secret。

## 定时执行示例

如果你想每月自动执行一次，可以在调用方仓库增加：

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
          lark-webhook-url: ${{ secrets.DNSHE_NOTIFY_LARK_WEBHOOK_URL }}
```
