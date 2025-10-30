# ログ設計書

## 1. 設計方針

本システムにおけるログは、AWS Lambdaで実行されることを前提とし、以下の目的を達成するために設計する。

-   **問題発生時の迅速な原因特定**: リクエスト単位での処理の流れを追跡可能にする。
-   **運用監視**: CloudWatch Logs Insightsやその他の監視ツールと連携し、エラー率やパフォーマンスの傾向を分析可能にする。

上記目的のため、以下の基本方針を採用する。

### 1.1. 構造化ロギング (JSON)

全てのログはJSON形式で標準出力に出力する。これにより、CloudWatch Logsが自動的にログをパースし、CloudWatch Logs Insightsで特定のフィールドに基づいた高度なクエリ（検索、集計、可視化）を実行できるようになる。

### 1.2. ログレベル

ログレベルを明確に定義し、Lambdaの環境変数 `LOG_LEVEL` で動的に出力レベルを制御できるようにする。これにより、本番環境ではパフォーマンスへの影響を最小限に抑えつつ、問題調査時には詳細なログを取得できる。

-   **ERROR**: 処理が続行不可能な致命的なエラー。
-   **WARN**: 処理は続行可能だが、想定外の事態が発生した場合。
-   **INFO**: 正常系の主要な処理の開始・終了など、処理の節目となる情報。
-   **DEBUG**: 開発・デバッグ時にのみ必要となる詳細情報。

## 2. 共通ログフィールド

すべてのログメッセージには、リクエストの追跡とコンテキストの把握を容易にするため、以下の共通フィールドを含める。

| フィールド名 | データ型 | 説明 |
| :--- | :--- | :--- |
| `level` | string | ログレベル（`error`, `warn`, `info`, `debug`）。`logrus`が自動で付与。 |
| `msg` | string | ログメッセージの本文。`logrus`が自動で付与。 |
| `time` | string | ログが出力されたタイムスタンプ（ISO 8601形式）。`logrus`が自動で付与。 |
| `aws_request_id` | string | Lambdaの実行を一意に識別するID。リクエストのトレースに必須。 |
| `function_name` | string | 実行されているLambda関数名。ログの発生源を特定する。 |

## 3. ログレベルごとの仕様と出力例

### 3.1. INFO

-   **目的**: 正常な処理フローの節目を確認する。
-   **例**: `extractor`関数がS3イベントをトリガーに処理を開始した時。

```json
{
  "aws_request_id": "a1b2c3d4-e5f6-7890-1234-567890abcdef",
  "function_name": "go-aws-handson-extractor",
  "level": "info",
  "msg": "Start extracting EXIF data",
  "s3_bucket": "my-image-bucket",
  "s3_key": "images/AkihabaraKousaten.jpeg",
  "time": "2025-10-30T14:00:00Z"
}
```

### 3.2. DEBUG

-   **目的**: 開発・デバッグ時に、処理の内部状態を詳細に確認する。
-   **例**: 抽出したメタデータの内容を確認する時。

```json
{
  "aws_request_id": "a1b2c3d4-e5f6-7890-1234-567890abcdef",
  "function_name": "go-aws-handson-extractor",
  "level": "debug",
  "metadata": {
    "Manufacturer": "Canon",
    "Model": "Canon EOS 7D",
    "DateTimeOriginal": "..."
  },
  "msg": "Successfully extracted metadata",
  "time": "2025-10-30T14:00:01Z"
}
```

### 3.3. WARN

-   **目的**: 予期せぬ状態だが、エラーとして処理を中断するほどではない事象を記録する。
-   **例**: 画像にEXIF情報が見つからなかった時。

```json
{
  "aws_request_id": "b2c3d4e5-f6a7-8901-2345-67890abcdef1",
  "function_name": "go-aws-handson-extractor",
  "level": "warning",
  "msg": "No EXIF data found in the image",
  "s3_bucket": "my-image-bucket",
  "s3_key": "images/no-exif.jpeg",
  "time": "2025-10-30T14:01:00Z"
}
```

### 3.4. ERROR

-   **目的**: 処理が失敗した原因を特定するための情報を記録する。
-   **例**: DynamoDBへのアイテム保存に失敗した時。

```json
{
  "aws_request_id": "c3d4e5f6-a7b8-9012-3456-7890abcdef12",
  "error": "ValidationException: One or more parameter values were invalid: ...",
  "function_name": "go-aws-handson-extractor",
  "level": "error",
  "msg": "Failed to save metadata to DynamoDB",
  "s3_key": "images/AkihabaraKousaten.jpeg",
  "time": "2025-10-30T14:02:00Z"
}
```

## 4. 実装

### 4.1. ライブラリ

ロギングライブラリとして、構造化ロギングとログレベルの制御に優れた `github.com/sirupsen/logrus` を採用する。

### 4.2. ログレベルの設定

ログレベルは、Lambdaの環境変数 `LOG_LEVEL` によって設定する。設定可能な値は `error`, `warn`, `info`, `debug` とする。環境変数が未設定、または不正な値の場合は `info` をデフォルトレベルとする。

### 4.3. 初期化コード例

各Lambda関数の `init()` 関数で以下の設定を行い、ロガーを初期化する。

```go
import (
    "os"
    "github.com/sirupsen/logrus"
)

func init() {
    // 出力形式をJSONに設定
    logrus.SetFormatter(&logrus.JSONFormatter{})

    // Lambda環境では標準出力がCloudWatch Logsに送られる
    logrus.SetOutput(os.Stdout)

    // 環境変数からログレベルを取得して設定
    level, err := logrus.ParseLevel(os.Getenv("LOG_LEVEL"))
    if err != nil {
        level = logrus.InfoLevel
    }
    logrus.SetLevel(level)
}
```
