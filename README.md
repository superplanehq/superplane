# SuperPlane

SuperPlane は、イベント駆動ワークフローを定義・実行するための **オープンソース DevOps コントロールプレーン** です。  
Git、CI/CD、可観測性、インシデント対応、インフラ、通知など、すでに利用しているツールをまたいで動作します。

![SuperPlane screenshot](./screenshot.png)

## プロジェクトの状況

<p>
  <a href="https://superplanehq.semaphoreci.com/projects/superplane"><img src="https://superplanehq.semaphoreci.com/badges/superplane/branches/main.svg?style=shields" alt="CI Status on Semaphore" /></a>
  <a href="https://github.com/superplanehq/superplane/pulse"><img src="https://img.shields.io/github/commit-activity/m/superplanehq/superplane" alt="GitHub commit activity"/></a>
  <a href="https://discord.gg/KC78eCNsnw"><img src="https://img.shields.io/discord/1409914582239023200?label=discord" alt="Discord server" /></a>
</p>

このプロジェクトはアルファ段階で、現在も急速に進化しています。  
コアモデルと統合機能を安定化している段階のため、粗い部分や破壊的変更が発生する可能性があります。  
試してみて分かりづらい点があれば、[Issue を作成](https://github.com/superplanehq/superplane/issues/new) してください。  
早期のフィードバックは非常に重要です。

## できること

- **ワークフローのオーケストレーション**: 複数システムをまたぐ多段の運用ワークフローをモデル化できます。
- **イベント駆動の自動化**: push、デプロイイベント、アラート、スケジュール、Webhook からワークフローを開始できます。
- **コントロールプレーン UI**: DevOps プロセスを設計・管理し、実行結果、ステータス、履歴を 1 か所で確認できます。
- **運用コンテキストの共有**: スクリプトが散在した状態ではなく、ワークフロー定義と運用意図を 1 つのシステムに集約できます。

## 仕組み

- **Canvases**: ワークフローを、ステップと依存関係で構成される有向グラフ（「Canvas」）としてモデル化します。
- **Components**: 各ステップは再利用可能なコンポーネント（組み込みまたは連携ベース）で、アクションを実行します（例: CI/CD 呼び出し、インシデント起票、通知送信、条件待機、承認必須化）。
- **イベントとトリガー**: 受信イベント（Webhook、スケジュール、各種ツールイベント）がトリガー条件に一致すると、イベントペイロードを入力として実行を開始します。
- **実行と可視化**: SuperPlane がグラフを実行して状態を追跡し、実行履歴やデバッグ情報を UI（および CLI）で提供します。

### 利用例

SuperPlane でチームが実際に構築している具体例:

- **ポリシーで保護された本番デプロイ**: CI が成功したら、営業時間外は保留し、オンコール + プロダクト承認を必須にしてからデプロイを実行。
- **段階的デリバリー (10% → 30% → 60% → 100%)**: 複数ウェーブでデプロイし、各ステップで待機・検証し、失敗時は承認ゲート付きでロールバック。
- **マルチリポジトリのリリーストレイン**: 複数サービスのタグ/ビルドを待機し、すべて揃ったら集約して協調デプロイを実行。
- **インシデント発生後「最初の5分」トリアージ**: インシデント作成時にコンテキスト（直近デプロイ + ヘルスシグナル）を並列取得し、証跡パックを生成して Issue を起票。

## クイックスタート

最新のデモコンテナを実行します:

```
docker pull ghcr.io/superplanehq/superplane-demo:stable
docker run --rm -p 3000:3000 -v spdata:/app/data -ti ghcr.io/superplanehq/superplane-demo:stable
```

その後 [http://localhost:3000](http://localhost:3000) を開き、[クイックスタートガイド](https://docs.superplane.com/get-started/quickstart/) に従ってください。

## サポートされている連携

SuperPlane はすでに利用しているツールと連携できます。  
各連携はトリガー（ワークフロー開始イベント）とコンポーネント（実行可能なアクション）を提供します。

> 連携一覧は [ドキュメント](https://docs.superplane.com/components/) を参照してください。追加してほしいプロバイダーがあれば、[Issue を作成](https://github.com/superplanehq/superplane/issues/new) してください。

### AI・LLM

<table>
<tr>
<td align="center" width="150"><a href="https://docs.superplane.com/components/claude/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/claude.svg" alt="Claude"/><br/>Claude</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/openai/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/openai.svg" alt="OpenAI"/><br/>OpenAI</a></td>
</tr>
</table>

### バージョン管理 & CI/CD

<table>
<tr>
<td align="center" width="150"><a href="https://docs.superplane.com/components/github/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/github.svg" alt="GitHub"/><br/>GitHub</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/gitlab/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/gitlab.svg" alt="GitLab"/><br/>GitLab</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/semaphore/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/semaphore-logo-sign-black.svg" alt="Semaphore"/><br/>Semaphore</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/render/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/render.svg" alt="Render"/><br/>Render</a></td>
</tr>
</table>

### クラウド & インフラストラクチャ

<table>
<tr>
<td align="center" width="150"><a href="https://docs.superplane.com/components/aws/#ecr-•-on-image-push" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/aws.ecr.svg" alt="AWS ECR"/><br/>AWS ECR</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/aws/#lambda-•-run-function" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/aws.lambda.svg" alt="AWS Lambda"/><br/>AWS Lambda</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/aws/#code-artifact-•-on-package-version" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/aws.codeartifact.svg" alt="AWS CodeArtifact"/><br/>AWS CodeArtifact</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/cloudflare/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/cloudflare.svg" alt="Cloudflare"/><br/>Cloudflare</a></td>
<td align="center" width="150"><a href="https://docs.superplane.com/components/dockerhub/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/docker.svg" alt="DockerHub"/><br/>DockerHub</a></td>
</tr>
</table>

### 可観測性

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

### 開発ツール

<table>
<tr>
<td align="center" width="150"><a href="https://docs.superplane.com/components/daytona/" target="_blank"><img width="40" src="https://raw.githubusercontent.com/superplanehq/superplane/main/web_src/src/assets/icons/integrations/daytona.svg" alt="Daytona"/><br/>Daytona</a></td>
</tr>
</table>

## 本番環境へのインストール

SuperPlane は単一ホストまたは Kubernetes にデプロイできます:

- **[単一ホストインストール](https://docs.superplane.com/installation/overview/#single-host-installation)** - AWS EC2、GCP Compute Engine、その他のクラウド環境にデプロイ
- **[Kubernetes インストール](https://docs.superplane.com/installation/overview/#kubernetes)** - GKE、EKS、または任意の Kubernetes クラスタにデプロイ

## ロードマップ概要

このセクションでは、SuperPlane が現在サポートしている内容と今後の予定を簡潔に示します。

**現在利用可能**

✓ 75+ 個のコンポーネント  
✓ イベント駆動ワークフローエンジン  
✓ ビジュアル Canvas ビルダー  
✓ 実行履歴、イベントチェーン表示、デバッグコンソール  
✓ スターター CLI とサンプルワークフロー

**進行中 / 今後予定**

→ 200+ の新規コンポーネント（AWS、Grafana、DataDog、Azure、GitLab、Jira ほか）  
→ [Canvas のバージョン管理](https://github.com/superplanehq/superplane/issues/1380)  
→ [SAML/SCIM](https://github.com/superplanehq/superplane/issues/1377) と [拡張 RBAC / 権限管理](https://github.com/superplanehq/superplane/issues/1378)  
→ [アーティファクトのバージョントラッキング](https://github.com/superplanehq/superplane/issues/1382)  
→ [公開 API](https://github.com/superplanehq/superplane/issues/1854)

## コントリビュート

バグ報告、改善アイデア、焦点を絞った PR を歓迎します。

- まずは **[コントリビューティングガイド](CONTRIBUTING.md)** を確認してください。
- バグ報告や機能要望は GitHub Issues を利用してください。

## ライセンス

Apache License 2.0. `LICENSE` を参照してください。

## コミュニティ

- **[Discord](https://discord.superplane.com)** - 議論、質問、コラボレーションのためのコミュニティに参加できます
- **[X](https://x.com/superplanehq)** - 最新情報やお知らせをフォローできます
