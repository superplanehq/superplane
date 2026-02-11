# SuperPlane

SuperPlaneは、イベントベースのワークフローを定義・実行するための**オープンソースDevOpsコントロールプレーン**です。
Git、CI/CD、オブザーバビリティ、インシデント対応、インフラ、通知など、既にお使いのツールと連携して動作します。

![SuperPlane スクリーンショット](./screenshot.png)

## プロジェクトの状況

<p>
  <a href="https://superplanehq.semaphoreci.com/projects/superplane"><img src="https://superplanehq.semaphoreci.com/badges/superplane/branches/main.svg?style=shields" alt="SemaphoreでのCIステータス" /></a>
  <a href="https://github.com/superplanehq/superplane/pulse"><img src="https://img.shields.io/github/commit-activity/m/superplanehq/superplane" alt="GitHubコミット活動"/></a>
  <a href="https://discord.gg/KC78eCNsnw"><img src="https://img.shields.io/discord/1409914582239023200?label=discord" alt="Discordサーバー" /></a>
</p>

本プロジェクトはアルファ段階にあり、急速に開発が進んでいます。コアモデルとインテグレーションの安定化に伴い、
粗削りな部分や破壊的変更が発生する可能性があります。
お試しいただいて分かりにくい点がございましたら、ぜひ[Issueを作成](https://github.com/superplanehq/superplane/issues/new)してください。
初期のフィードバックは非常に貴重です。

## 主な機能

- **ワークフローオーケストレーション**: 複数のシステムにまたがるマルチステップの運用ワークフローをモデル化します。
- **イベント駆動の自動化**: プッシュ、デプロイイベント、アラート、スケジュール、Webhookからワークフローをトリガーします。
- **コントロールプレーンUI**: DevOpsプロセスを設計・管理し、実行状況、ステータス、履歴を一つの画面で確認できます。
- **共有された運用コンテキスト**: 散在するスクリプトの代わりに、ワークフロー定義と運用意図を一つのシステムにまとめます。

## 仕組み

- **キャンバス**: ワークフローをステップと依存関係からなる有向グラフ（「キャンバス」）としてモデル化します。
- **コンポーネント**: 各ステップはアクションを実行する再利用可能なコンポーネント（組み込みまたはインテグレーション連携）です（例: CI/CDの呼び出し、インシデントの作成、通知の送信、条件の待機、承認の要求）。
- **イベントとトリガー**: 受信イベント（Webhook、スケジュール、ツールイベント）がトリガーにマッチすると、イベントペイロードを入力として実行が開始されます。
- **実行と可視化**: SuperPlaneがグラフを実行し、状態を追跡し、UI（およびCLI）で実行状況・履歴・デバッグ情報を表示します。

### ユースケースの例

SuperPlaneを使ってチームが構築する具体例をいくつかご紹介します:

- **ポリシーゲート付き本番デプロイ**: CIがグリーンで完了したら、営業時間外は保留し、オンコール担当者とプロダクトの承認を要求した後、デプロイをトリガーします。
- **プログレッシブデリバリー（10% → 30% → 60% → 100%）**: 段階的にデプロイし、各ステップで待機・検証を行い、承認ゲート付きで失敗時にはロールバックします。
- **マルチリポジトリのリリーストレイン**: 一連のサービスからのタグ/ビルドを待ち、すべて準備完了したらファンインし、協調したデプロイを実行します。
- **「最初の5分間」インシデントトリアージ**: インシデント作成時に、コンテキスト（最近のデプロイ＋ヘルスシグナル）を並行取得し、エビデンスパックを生成してIssueを作成します。

## クイックスタート

最新のデモコンテナを実行します:

```
docker pull ghcr.io/superplanehq/superplane-demo:stable
docker run --rm -p 3000:3000 -v spdata:/app/data -ti ghcr.io/superplanehq/superplane-demo:stable
```

[http://localhost:3000](http://localhost:3000) を開き、[クイックスタートガイド](https://docs.superplane.com/get-started/quickstart/)に従ってください。

## 対応インテグレーション

SuperPlaneは既にお使いのツールと統合できます。各インテグレーションはトリガー（ワークフローを開始するイベント）とコンポーネント（実行可能なアクション）を提供します。

> 完全なリストは[ドキュメント](https://docs.superplane.com/components/)をご覧ください。必要なプロバイダーがない場合は、[Issueを作成](https://github.com/superplanehq/superplane/issues/new)してリクエストしてください。

### AI・LLM

<table>
<tr>
<td align="center" width="150"><a href="https://docs.superplane.com/components/claude/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/claude.svg" alt="Claude"/><br/>Claude</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/openai/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/openai.svg" alt="OpenAI"/><br/>OpenAI</a></td>
</tr>
</table>

### バージョン管理・CI/CD

<table>
<tr>
<td align="center" width="150"><a href="https://docs.superplane.com/components/github/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/github.svg" alt="GitHub"/><br/>GitHub</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/gitlab/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/gitlab.svg" alt="GitLab"/><br/>GitLab</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/semaphore/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/semaphore-logo-sign-black.svg" alt="Semaphore"/><br/>Semaphore</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/render/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/render.svg" alt="Render"/><br/>Render</a></td>
</tr>
</table>

### クラウド・インフラ

<table>
<tr>
<td align="center" width="150"><a href="https://docs.superplane.com/components/aws/#ecr-•-on-image-push" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/aws.ecr.svg" alt="AWS ECR"/><br/>AWS ECR</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/aws/#lambda-•-run-function" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/aws.lambda.svg" alt="AWS Lambda"/><br/>AWS Lambda</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/aws/#code-artifact-•-on-package-version" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/aws.codeartifact.svg" alt="AWS CodeArtifact"/><br/>AWS CodeArtifact</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/cloudflare/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/cloudflare.svg" alt="Cloudflare"/><br/>Cloudflare</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/dockerhub/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/docker.svg" alt="DockerHub"/><br/>DockerHub</a></td>
</tr>
</table>

### オブザーバビリティ

<table>
<tr>
<td align="center" width="150"><a href="https://docs.superplane.com/components/datadog/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/datadog.svg" alt="DataDog"/><br/>DataDog</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/dash0/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/dash0.svg" alt="Dash0"/><br/>Dash0</a></td>
</tr>
</table>

### インシデント管理

<table>
<tr>
<td align="center" width="150"><a href="https://docs.superplane.com/components/pagerduty/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/pagerduty.svg" alt="PagerDuty"/><br/>PagerDuty</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/rootly/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/rootly.svg" alt="Rootly"/><br/>Rootly</a></td>
</tr>
</table>

### コミュニケーション

<table>
<tr>
<td align="center" width="150"><a href="https://docs.superplane.com/components/discord/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/discord.svg" alt="Discord"/><br/>Discord</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/slack/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/slack.svg" alt="Slack"/><br/>Slack</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/sendgrid/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/sendgrid.svg" alt="SendGrid"/><br/>SendGrid</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/smtp/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/smtp.svg" alt="SMTP"/><br/>SMTP</a></td>
</tr>
</table>

### チケット管理

<table>
<tr>
<td align="center" width="150"><a href="https://docs.superplane.com/components/jira/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/jira.svg" alt="Jira"/><br/>Jira</a></td>
</tr>
</table>

### 開発者ツール

<table>
<tr>
<td align="center" width="150"><a href="https://docs.superplane.com/components/daytona/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/daytona.svg" alt="Daytona"/><br/>Daytona</a></td>
</tr>
</table>

## 本番環境へのインストール

SuperPlaneは単一ホストまたはKubernetes上にデプロイできます:

- **[単一ホストインストール](https://docs.superplane.com/installation/overview/#single-host-installation)** - AWS EC2、GCP Compute Engine、その他のクラウドプロバイダーにデプロイ
- **[Kubernetesインストール](https://docs.superplane.com/installation/overview/#kubernetes)** - GKE、EKS、またはその他のKubernetesクラスターにデプロイ

## ロードマップ概要

このセクションでは、SuperPlaneが現在サポートしている機能と今後の予定を簡単にご紹介します。

**現在利用可能**

✓ 75以上のコンポーネント
✓ イベント駆動ワークフローエンジン
✓ ビジュアルキャンバスビルダー
✓ 実行履歴、イベントチェーンビュー、デバッグコンソール
✓ スターターCLIとサンプルワークフロー

**開発中・今後の予定**

→ 200以上の新コンポーネント（AWS、Grafana、DataDog、Azure、GitLab、Jiraなど）
→ [キャンバスのバージョン管理](https://github.com/superplanehq/superplane/issues/1380)
→ [SAML/SCIM](https://github.com/superplanehq/superplane/issues/1377)および[拡張RBACと権限管理](https://github.com/superplanehq/superplane/issues/1378)
→ [アーティファクトバージョン追跡](https://github.com/superplanehq/superplane/issues/1382)
→ [パブリックAPI](https://github.com/superplanehq/superplane/issues/1854)

## コントリビューション

バグレポート、改善のアイデア、的を絞ったPRを歓迎します。

- はじめに**[コントリビューションガイド](CONTRIBUTING.md)**をお読みください。
- Issue: バグや機能リクエストにはGitHub Issueをご利用ください。

## ライセンス

Apache License 2.0。詳細は `LICENSE` をご覧ください。

## コミュニティ

- **[Discord](https://discord.superplane.com)** - ディスカッション、質問、コラボレーションのためのコミュニティにご参加ください
- **[X](https://x.com/superplanehq)** - 最新情報やお知らせはこちらをフォローしてください
