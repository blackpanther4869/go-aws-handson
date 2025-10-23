# Lambda 設計書

## 1. 概要

本システムのビジネスロジックは、3つのAWS Lambda関数によって実装される。
各関数はGo言語で実装され、単一責任の原則に基づき、それぞれ特定の役割を担う。

## 2. 関数一覧

| 関数名 | トリガー | 役割 |
|:---|:---|:---|
| 署名付きURL発行Lambda | API Gateway | S3の署名付きURLを発行する。 |
| メタデータ抽出Lambda | S3 (PutObject) | 画像のExifメタデータを抽出し、DynamoDBに保存する。 |
| メタデータ検索Lambda | API Gateway | DynamoDBからメタデータを検索し、結果を返却する。 |

## 3. 共通仕様

- **ランタイム**: `go1.x`
- **アーキテクチャ**: `arm64`
- **ロギング**: `go.uber.org/zap` を使用した構造化ロギング。ログはCloudWatch Logsに出力される。
- **環境変数**:
    - `LOG_LEVEL`: ログレベル (`DEBUG`, `INFO`, `WARN`, `ERROR`) を制御する。
    - `DYNAMODB_TABLE_NAME`: 操作対象のDynamoDBテーブル名。
    - `S3_BUCKET_NAME`: 操作対象のS3バケット名。

## 4. 各関数詳細

### 4.1. 署名付きURL発行Lambda

- **メモリ**: 128MB
- **タイムアウト**: 10秒
- **IAMロール権限**:
    - `s3:PutObject` (特定のS3バケットの `uploads/*` プレフィックスに対して)
    - `logs:CreateLogGroup`, `logs:CreateLogStream`, `logs:PutLogEvents`
- **処理フロー**:
    1. API Gatewayから `fileName` と `contentType` を受け取る。
    2. S3の `PresignClient` を使用して、`PutObject` 操作のための署名付きURLを生成する。
    3. 生成した署名付きURLと、アップロード後の画像URLをレスポンスとして返却する。

### 4.2. メタデータ抽出Lambda

- **メモリ**: 256MB (画像処理のため多めに設定)
- **タイムアウト**: 30秒
- **IAMロール権限**:
    - `s3:GetObject` (特定のS3バケットの `uploads/*` プレフィックスに対して)
    - `dynamodb:PutItem` (特定のDynamoDBテーブルに対して)
    - `logs:CreateLogGroup`, `logs:CreateLogStream`, `logs:PutLogEvents`
- **処理フロー**:
    1. S3のPutObjectイベントをトリガーに起動する。
    2. イベント情報からバケット名とオブジェクトキーを取得する。
    3. S3から対象の画像オブジェクトをダウンロードする。
    4. `goexif/exif` ライブラリを使用してExifメタデータを抽出する。
    5. 抽出したメタデータと、ファイルサイズなどの基本情報を合わせて、DynamoDBに保存する。

### 4.3. メタデータ検索Lambda

- **メモリ**: 128MB
- **タイムアウト**: 10秒
- **IAMロール権限**:
    - `dynamodb:GetItem`, `dynamodb:Query` (特定のDynamoDBテーブルおよびGSIに対して)
    - `logs:CreateLogGroup`, `logs:CreateLogStream`, `logs:PutLogEvents`
- **処理フロー**:
    1. API Gatewayからパスパラメータ (`imageID`) またはクエリパラメータ (`fileName`, `from`, `to`) を受け取る。
    2. パラメータに応じて、DynamoDBの `GetItem` または `Query` 操作を実行する。
    3. 取得した検索結果をクライアントに返却する。