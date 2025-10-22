# 🚀 AWS構築＆Go開発ハンズオン - 詳細手順書

## はじめに

このドキュメントは、「Go & Serverless: GitHub Actionsで構築する画像メタデータ・インスペクター」ハンズオンの詳細な作業手順書です。Java開発経験数年の中級者レベルで、Go言語およびAWSの利用は初心者の方を対象としています。

本ハンズオンでは、Gemini CLIをドキュメント作成、コード生成、デバッグ、テスト実行など、開発の様々な局面で活用しながら、以下のシステムを構築します。

**システム概要**: S3にアップロードされた画像（JPEG/TIFF）からGo LambdaがExifメタデータを抽出し、DynamoDBに永続化するシステムを構築する。

## 1. 環境セットアップ

Go言語、Docker、AWS CLI、AWS CDK CLI、Gemini CLIの準備を行います。

### 1.1. Go言語のインストール

最新のLTSバージョン（例: Go 1.21.x）を推奨します。環境構築を必要最低限に抑えるため、公式インストーラまたは`goenv`などのバージョン管理ツールを利用します。

1.  **公式インストーラ**: Goの公式サイトからOSに合ったインストーラをダウンロードし、指示に従ってインストールします。
2.  **goenv (推奨)**: 複数のGoバージョンを管理する場合に便利です。
    ```bash
    git clone https://github.com/go-nv/goenv.git ~/.goenv
    echo 'export PATH="$HOME/.goenv/bin:$PATH"' >> ~/.bash_profile # または ~/.zshrc
    echo 'eval "$(goenv init -)"' >> ~/.bash_profile # または ~/.zshrc
    source ~/.bash_profile # または ~/.zshrc

    goenv install latest
    goenv global latest
    go version
    ```

### 1.2. Dockerのインストール

MinIOおよびDynamoDB LocalをローカルでエミュレートするためにDockerを使用します。Docker Desktopを公式サイトからダウンロードし、インストールしてください。

*   [Docker Desktop](https://www.docker.com/products/docker-desktop/)

### 1.3. AWS CLIのインストールと初期設定

AWSリソースの手動操作や、AWS CDKの認証情報としてAWS CLIを使用します。

1.  **インストール**: 公式ドキュメントに従ってAWS CLIをインストールします。
    *   [AWS CLI のインストール](https://docs.aws.amazon.com/ja_jp/cli/latest/userguide/getting-started-install.html)
2.  **初期設定**: AWSアカウントの認証情報を設定します。
    ```bash
    aws configure
    ```
    *   `AWS Access Key ID`: AWSアカウントのアクセスキーIDを入力
    *   `AWS Secret Access Key`: AWSアカウントのシークレットアクセスキーIDを入力
    *   `Default region name`: `ap-northeast-1`など、使用するAWSリージョンを入力
    *   `Default output format`: `json`を入力

### 1.4. AWS CDK CLIのインストール

AWS CDK (Go) を使用してIaCを記述・デプロイするために、AWS CDK CLIをインストールします。

```bash
npm install -g aws-cdk
cdk --version
```

### 1.5. Gemini CLIのインストール

Gemini CLIは既にセットアップ済みです。

## 2. ローカルエミュレーション環境の構築

MinIO (S3代替) および DynamoDB LocalをDockerで起動し、ローカル開発環境を構築します。

### 2.1. `docker-compose.yml`の作成

プロジェクトルートに`docker-compose.yml`ファイルを作成し、以下の内容を記述します。

```yaml
version: '3.8'

services:
  minio:
    image: minio/minio
    ports:
      - "9000:9000"
      - "9001:9001"
    environment:
      MINIO_ROOT_USER: minioadmin
      MINIO_ROOT_PASSWORD: minioadmin
    command: server /data --console-address ":9001"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:9000/minio/health/live"]
      interval: 30s
      timeout: 20s
      retries: 3

  dynamodb-local:
    image: amazon/dynamodb-local
    ports:
      - "8000:8000"
    command: "-jar DynamoDBLocal.jar -sharedDb -dbPath /home/dynamodblocal/data"
    volumes:
      - ./data:/home/dynamodblocal/data
    working_dir: /home/dynamodblocal
```

### 2.2. ローカルエミュレータの起動

プロジェクトルートで以下のコマンドを実行し、MinIOとDynamoDB Localを起動します。

```bash
docker-compose up -d
```

起動後、以下のURLで各サービスにアクセスできることを確認します。

*   MinIO Console: `http://localhost:9001` (ユーザー: `minioadmin`, パスワード: `minioadmin`)
*   DynamoDB Local: `http://localhost:8000` (AWS CLIなどでアクセス)

## 3. Goプロジェクトの初期化とフォルダ構成

Goのベストプラクティス (`cmd`, `internal`, `pkg`) に従ったプロジェクト構造を作成します。

### 3.1. プロジェクトの初期化

プロジェクトルートで以下のコマンドを実行し、Goモジュールを初期化します。

```bash
go mod init github.com/your-github-username/go-aws-handson # ご自身のGitHubユーザー名に置き換えてください
```

### 3.2. フォルダ構成の作成

以下のフォルダ構成を作成します。

```
.
├── cmd/
│   ├── get-signed-url/ # 署名付きURL発行Lambdaのハンドラ
│   ├── extract-metadata/ # メタデータ抽出Lambdaのハンドラ
│   └── search-metadata/ # メタデータ検索Lambdaのハンドラ
├── internal/
│   ├── exif/ # Exif抽出ロジック
│   ├── repository/ # DynamoDBリポジトリ層
│   └── util/ # 共通ユーティリティ
├── pkg/ # 外部公開可能な共通ライブラリ（今回は使用しない可能性あり）
├── docs/ # ドキュメント類
├── .github/ # GitHub Actionsワークフロー
├── data/ # DynamoDB Localのデータ保存用
├── docker-compose.yml
├── go.mod
└── go.sum
```

## 4. Go開発 - メタデータ抽出ロジックの実装と単体テスト

`github.com/rwcarlsen/goexif/exif`ライブラリを使用して、画像からExifメタデータを抽出するロジックを実装します。

### 4.1. Exif抽出ライブラリのインストール

```bash
go get github.com/rwcarlsen/goexif/exif
```

### 4.2. Exif抽出ロジックの実装

`internal/exif/extractor.go`にExif抽出ロジックを実装します。

### 4.3. 単体テストの記述と実行

`internal/exif/extractor_test.go`に単体テストを記述し、`go test ./internal/exif`で実行します。

## 5. Go開発 - DynamoDBリポジトリ層の実装と単体テスト

抽出したExifメタデータをDynamoDBに保存・取得するためのリポジトリ層を実装します。

### 5.1. AWS SDK for Go v2のインストール

```bash
go get github.com/aws/aws-sdk-go-v2
go get github.com/aws/aws-sdk-go-v2/config
go get github.com/aws/aws-sdk-go-v2/service/dynamodb
go get github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue
```

### 5.2. DynamoDBリポジトリ層の実装

`internal/repository/metadata_repository.go`にメタデータ保存・取得ロジックを実装します。

### 5.3. 単体テストの記述と実行

`internal/repository/metadata_repository_test.go`に単体テストを記述し、DynamoDB Localに対してテストを実行します。

## 6. Go開発 - Lambdaハンドラの実装と統合テスト

3つのLambda関数（署名付きURL発行、メタデータ抽出、メタデータ検索）のハンドラを実装し、ローカルで統合テストを行います。

### 6.1. Lambdaランタイムライブラリのインストール

```bash
go get github.com/aws/aws-lambda-go/lambda
go get github.com/aws/aws-lambda-go/events
```

### 6.2. 署名付きURL発行Lambdaハンドラの実装

`cmd/get-signed-url/main.go`にAPI Gatewayイベントをハンドリングし、S3署名付きPUT URLを生成するロジックを実装します。

### 6.3. メタデータ抽出Lambdaハンドラの実装

`cmd/extract-metadata/main.go`にS3イベントをハンドリングし、S3からの画像ダウンロード、Exif抽出、DynamoDB永続化を連携するロジックを実装します。

### 6.4. メタデータ検索Lambdaハンドラの実装

`cmd/search-metadata/main.go`にAPI Gatewayイベントをハンドリングし、DynamoDBからのメタデータ検索ロジックを実装します。

### 6.5. ローカルでの統合テスト

AWS SAM CLIを使用して、Lambdaハンドラのローカルでの動作確認を行います。

1.  **AWS SAM CLIのインストール**: 公式ドキュメントに従ってインストールします。
    *   [AWS SAM CLI のインストール](https://docs.aws.amazon.com/ja_jp/serverless-application-model/latest/developerguide/install-samcli.html)
2.  **`template.yaml`の作成**: 各Lambda関数の定義を記述します。
3.  **ローカル実行**: `sam local invoke`コマンドを使用して、各Lambda関数をローカルで実行し、動作を確認します。

## 7. AWSリソースの手動定義とIAMポリシーの指針

AWSマネジメントコンソールまたはAWS CLIを使用して、S3バケット、DynamoDBテーブル、API Gatewayなどのリソースを手動で作成します。このステップでは、IAMポリシーの最小権限の原則に基づいた具体的な記述指針も示します。

### 7.1. S3バケットの作成

*   バケット名: `go-aws-handson-image-bucket-{your-account-id}` (一意になるようにアカウントIDなどを付与)
*   パブリックアクセスブロック設定: 全てブロック
*   バージョン管理: 無効
*   イベント通知: メタデータ抽出Lambdaをトリガーするように設定（`PutObject`イベント、プレフィックス`uploads/`）

### 7.2. DynamoDBテーブルの作成

*   テーブル名: `ImageMetadata`
*   プライマリキー: `ImageID` (String)
*   グローバルセカンダリインデックス (GSI):
    *   `FileName-index` (パーティションキー: `FileName`, プロジェクション: ALL)
    *   `UploadTimestamp-index` (パーティションキー: `UploadTimestamp`, プロジェクション: ALL)
*   キャパシティモード: オンデマンド

### 7.3. IAMロールの作成とポリシーの指針

各Lambda関数用にIAMロールを作成し、以下の指針に基づいて最小限の権限を付与します。

1.  **署名付きURL発行Lambda用IAMロール**:
    *   **信頼ポリシー**: `lambda.amazonaws.com`からの引き受けを許可
    *   **アクセス権限ポリシー**:
        *   `s3:PutObject` (特定のS3バケットの`uploads/*`プレフィックスに対して)
        *   `logs:CreateLogGroup`, `logs:CreateLogStream`, `logs:PutLogEvents` (CloudWatch Logsへの書き込み)
2.  **メタデータ抽出Lambda用IAMロール**:
    *   **信頼ポリシー**: `lambda.amazonaws.com`からの引き受けを許可
    *   **アクセス権限ポリシー**:
        *   `s3:GetObject` (特定のS3バケットの`uploads/*`プレフィックスに対して)
        *   `dynamodb:PutItem` (特定のDynamoDBテーブルに対して)
        *   `logs:CreateLogGroup`, `logs:CreateLogStream`, `logs:PutLogEvents` (CloudWatch Logsへの書き込み)
3.  **メタデータ検索Lambda用IAMロール**:
    *   **信頼ポリシー**: `lambda.amazonaws.com`からの引き受けを許可
    *   **アクセス権限ポリシー**:
        *   `dynamodb:GetItem`, `dynamodb:Query` (特定のDynamoDBテーブルに対して)
        *   `logs:CreateLogGroup`, `logs:CreateLogStream`, `logs:PutLogEvents` (CloudWatch Logsへの書き込み)

### 7.4. API Gatewayの作成

*   REST APIを作成し、以下のリソースとメソッドを定義します。
    *   `/signed-url` (GET): 署名付きURL発行Lambdaと統合
    *   `/metadata/{imageID}` (GET): メタデータ検索Lambdaと統合
    *   `/metadata` (GET): メタデータ検索Lambdaと統合 (クエリパラメータ`fileName`, `from`, `to`を使用)
*   デプロイステージを作成し、APIを公開します。

## 8. AWS CDK (Go) によるIaC化

手動で作成したAWSリソースを、AWS CDK (Go) を使用してIaC化します。

### 8.1. AWS CDKプロジェクトの初期化

プロジェクトルートで以下のコマンドを実行し、AWS CDKプロジェクトを初期化します。

```bash
cdk init app --language go
```

### 8.2. リソース定義のGoコード記述

`main.go`や`stack.go`などのファイルに、S3バケット、DynamoDBテーブル、Lambda関数、API Gatewayなどのリソース定義をGoコードで記述します。

### 8.3. デプロイ

```bash
cdk deploy
```

### 8.4. 手動リソースの削除

`cdk deploy`が成功したら、手動で作成したAWSリソースは削除します。

## 9. CI/CD構築 (GitHub Actions)

GitHub Actionsを使用して、Goのビルド、テスト、AWS CDKによるデプロイを自動化するCI/CDパイプラインを構築します。

### 9.1. GitHubリポジトリの作成とコードのプッシュ

GitHubに新しいリポジトリを作成し、ローカルのプロジェクトコードをプッシュします。

### 9.2. `.github/workflows/ci-cd.yml`の作成

`.github/workflows/ci-cd.yml`ファイルを作成し、以下のステップを含むワークフローを記述します。

*   Goのセットアップ
*   依存関係のインストール
*   Goのビルド
*   Goのテスト
*   AWS認証 (OIDC認証を使用)
*   AWS CDKによるデプロイ (`cdk deploy`)

### 9.3. AWS OIDC認証の設定

GitHub ActionsからAWSリソースに安全にアクセスするために、AWS OIDC認証を設定します。

### 9.4. ワークフローの実行と確認

コードをGitHubにプッシュし、GitHub Actionsワークフローが自動的に実行され、AWSリソースがデプロイされることを確認します。

## 10. AWS環境での動作確認

デプロイされたシステムがAWS環境で正しく動作することを確認します。

### 10.1. エンドツーエンドのテストシナリオ

1.  API Gateway経由で署名付きURL発行APIを呼び出し、S3署名付きPUT URLを取得します。
2.  取得した署名付きURLを使用して、S3バケットに画像をアップロードします。
3.  数秒後、メタデータ検索APIを呼び出し、アップロードした画像のExifメタデータが取得できることを確認します。

### 10.2. CloudWatch Logsでのログ確認

各Lambda関数のCloudWatch Logsグループにアクセスし、実行ログやエラーログを確認します。

### 10.3. DynamoDBコンソールでのデータ確認

DynamoDBコンソールで`ImageMetadata`テーブルにアクセスし、保存されたExifメタデータを確認します。

## 11. リソースクリーンアップ

ハンズオン終了後、不要な課金を避けるため、作成した全てのAWSリソースを完全に削除します。

1.  **AWS CDKによるリソース削除**:
    ```bash
    cdk destroy
    ```
2.  **S3バケットの手動削除**:
    `cdk destroy`ではS3バケットが削除されない場合があるため、S3コンソールから手動でバケットを空にし、削除します。
3.  **IAMロールの手動削除**:
    Lambda関数に関連付けられたIAMロールは`cdk destroy`で削除されない場合があるため、IAMコンソールから手動で削除します。
4.  **CloudWatch Logsグループの手動削除**:
    Lambda関数に関連付けられたCloudWatch Logsグループは`cdk destroy`で削除されない場合があるため、CloudWatch Logsコンソールから手動で削除します。
