# 🚀 AWS構築＆Go開発ハンズオン - 詳細手順書

## はじめに

このドキュメントは、「Go & Serverless: GitHub Actionsで構築する画像メタデータ・インスペクター」ハンズオンの詳細な作業手順書です。Java開発経験数年の中級者レベルで、Go言語およびAWSの利用は初心者の方を対象としています。

本ハンズオンでは、Gemini CLIをドキュメント作成、コード生成、デバッグ、テスト実行など、開発の様々な局面で活用しながら、以下のシステムを構築します。

**システム概要**: S3にアップロードされた画像（JPEG/TIFF）からGo LambdaがExifメタデータを抽出し、DynamoDBに永続化するシステムを構築する。

## 1. 環境セットアップ

Go言語、Docker、AWS CLI、AWS CDK CLI、Gemini CLIの準備を行います。

### 1.1. Go言語のインストール

Homebrewを使用してGo言語をインストールします。最新のLTSバージョンがインストールされます。

```bash
brew install go
go version
```

### 1.2. AWSアカウントとIAMユーザーの作成

本ハンズオンではAWSリソースを操作するため、AWSアカウントが必要です。

1.  **AWSアカウントの作成**:
    *   [AWS公式サイト](https://aws.amazon.com/jp/)にアクセスし、画面の指示に従ってアカウントを作成します。
    *   クレジットカードの登録が必要ですが、本ハンズオンで利用するサービスは無料利用枠の範囲内です。
    *   アカウント作成後、まずは**ルートユーザー**（登録したメールアドレスとパスワード）でAWSマネジメントコンソールにサインインします。

2.  **IAMユーザーの作成**:
    *   日常的な開発作業をルートユーザーで行うことは推奨されません。代わりに、管理者権限を持つIAMユーザーを作成します。
    *   [IAMコンソール](https://console.aws.amazon.com/iam/)にアクセスします。
    *   左側のナビゲーションペインで **[ユーザー]** を選択し、**[ユーザーを追加]** をクリックします。
    *   **ユーザー名**（例: `handson-admin`）を入力し、**[AWS マネジメントコンソールへのアクセス]** にチェックを入れます。
    *   **[次のステップ: 許可]** をクリックします。
    *   **[既存のポリシーを直接アタッチ]** を選択し、ポリシーの一覧から `AdministratorAccess` を検索してチェックを入れます。
    *   **[次のステップ: タグ]**、**[次のステップ: レビュー]** と進み、内容を確認して **[ユーザーの作成]** をクリックします。

3.  **アクセスキーの作成**:
    *   作成したIAMユーザーの詳細画面を開き、**[認証情報]** タブを選択します。
    *   **[アクセスキーの作成]** をクリックし、ユースケースとして **[コマンドラインインターフェイス (CLI)]** を選択します。
    *   確認のチェックボックスにチェックを入れ、**[次のステップ]** をクリックします。
    *   **[アクセスキーを作成]** をクリックすると、**アクセスキーID**と**シークレットアクセスキー**が表示されます。
    *   **重要**: この画面を閉じるとシークレットアクセスキーは二度と表示できません。必ず `.csv` ファイルをダウンロードするか、表示されたキーを安全な場所にコピーして保存してください。このキーは後の `aws configure` で使用します。

### 1.3. Dockerのインストール

MinIOおよびDynamoDB LocalをローカルでエミュレートするためにDockerを使用します。Docker Desktopを公式サイトからダウンロードし、インストールしてください。

*   [Docker Desktop](https://www.docker.com/products/docker-desktop/)

### 1.4. AWS CLIのインストールと初期設定

AWSリソースの手動操作や、AWS CDKの認証情報としてAWS CLIを使用します。

1.  **インストール**: 公式ドキュメントに従ってAWS CLIをインストールします。
    *   [AWS CLI のインストール](https://docs.aws.amazon.com/ja_jp/cli/latest/userguide/getting-started-install.html)
2.  **初期設定**: AWSアカウントの認証情報を設定します。
    ```bash
    aws configure
    ```
    *   `AWS Access Key ID`: 先ほど作成したIAMユーザーのアクセスキーIDを入力
    *   `AWS Secret Access Key`: 先ほど作成したIAMユーザーのシークレットアクセスキーを入力
    *   `Default region name`: `ap-northeast-1`など、使用するAWSリージョンを入力
    *   `Default output format`: `json`を入力

### 1.5. AWS CDK CLIのインストール

AWS CDK (Go) を使用してIaCを記述・デプロイするために、AWS CDK CLIをインストールします。

```bash
npm install -g aws-cdk
cdk --version
```

### 1.6. Gemini CLIのインストール

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

### 2.3. ローカルエミュレータの動作確認

AWS CLIを使用して、各エミュレータが正常に動作していることを確認します。

ローカルエミュレータへのアクセスには、`--endpoint-url`の指定が必要です。また、AWSアカウントの認証情報が誤って使われるのを防ぐため、コマンド実行時に環境変数を指定して、各エミュレータ用のダミーの認証情報を渡すのが安全です。

#### 2.3.1. MinIO (S3) の動作確認

MinIOには、`docker-compose.yml`で設定したユーザー(`minioadmin`)とパスワード(`minioadmin`)を環境変数で指定します。

1.  **バケットの作成 (`mb`)**

    ```bash
    AWS_ACCESS_KEY_ID=minioadmin AWS_SECRET_ACCESS_KEY=minioadmin \
    aws s3 mb s3://test-bucket --endpoint-url http://localhost:9000
    ```

2.  **テストファイルの作成とアップロード (`cp`)**

    ```bash
    # テスト用のファイルを作成
    echo "hello minio" > dummy.txt

    # 作成したファイルを test-bucket にアップロード
    AWS_ACCESS_KEY_ID=minioadmin AWS_SECRET_ACCESS_KEY=minioadmin \
    aws s3 cp dummy.txt s3://test-bucket/ --endpoint-url http://localhost:9000
    ```

3.  **ファイル一覧の確認 (`ls`)**

    ```bash
    # test-bucket の中身を一覧表示
    AWS_ACCESS_KEY_ID=minioadmin AWS_SECRET_ACCESS_KEY=minioadmin \
    aws s3 ls s3://test-bucket/ --endpoint-url http://localhost:9000
    ```

    `dummy.txt`が表示されれば、アップロードは成功しています。

#### 2.3.2. DynamoDB Local の動作確認

DynamoDB Localは認証を検証しませんが、AWS CLIは認証情報とリージョンを要求します。そのため、ダミーの認証情報と任意のリージョン（例: `us-east-1`）を指定します。

1.  **テスト用のテーブル定義ファイルを作成**

    プロジェクトのルートに `test-table.json` という名前で以下のファイルを作成します。

    **`test-table.json`**
    ```json
    {
        "TableName": "TestTable",
        "AttributeDefinitions": [
            {
                "AttributeName": "id",
                "AttributeType": "S"
            }
        ],
        "KeySchema": [
            {
                "AttributeName": "id",
                "KeyType": "HASH"
            }
        ],
        "ProvisionedThroughput": {
            "ReadCapacityUnits": 1,
            "WriteCapacityUnits": 1
        }
    }
    ```

2.  **テーブルの作成 (`create-table`)**

    ```bash
    AWS_ACCESS_KEY_ID=dummy AWS_SECRET_ACCESS_KEY=dummy \
    aws dynamodb create-table \
        --cli-input-json file://test-table.json \
        --endpoint-url http://localhost:8000 \
        --region us-east-1
    ```

3.  **テーブル一覧の確認 (`list-tables`)**

    ```bash
    AWS_ACCESS_KEY_ID=dummy AWS_SECRET_ACCESS_KEY=dummy \
    aws dynamodb list-tables --endpoint-url http://localhost:8000 --region us-east-1
    ```
    `"TableNames": ["TestTable"]` と表示されれば成功です。

4.  **データの追加 (`put-item`)**

    ```bash
    AWS_ACCESS_KEY_ID=dummy AWS_SECRET_ACCESS_KEY=dummy \
    aws dynamodb put-item \
        --table-name TestTable \
        --item '{"id": {"S": "test-id-1"}, "message": {"S": "Hello, DynamoDB Local!"}}' \
        --endpoint-url http://localhost:8000 \
        --region us-east-1
    ```

5.  **データの確認 (`get-item`)**

    ```bash
    AWS_ACCESS_KEY_ID=dummy AWS_SECRET_ACCESS_KEY=dummy \
    aws dynamodb get-item \
        --table-name TestTable \
        --key '{"id": {"S": "test-id-1"}}' \
        --endpoint-url http://localhost:8000 \
        --region us-east-1
    ```
    追加したデータが表示されれば、DynamoDB Localも正常に動作しています。

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

`github.com/dsoprea/go-exif/v3`ライブラリを使用して、画像からExifメタデータを抽出するロジックを実装します。

### 4.1. Exif抽出ライブラリのインストール

プロジェクトのルートディレクトリで以下のコマンドを実行して、ライブラリとテスト用のアサーションライブラリをインストールします。

```bash
go get github.com/dsoprea/go-exif/v3
go get github.com/stretchr/testify/assert
```

### 4.2. ロジックの実装 (`internal/exif/extractor.go`)

`internal/exif/`ディレクトリに`extractor.go`ファイルを作成し、以下の内容を記述します。`io.Reader`からストリーミングでEXIFデータを読み込み、タグを一つずつ処理することで、メモリ効率の良い実装を目指します。

```go
package exif

import (
	"fmt"
	"io"
	"log"
	"time"

	"github.com/dsoprea/go-exif/v3"
	exifcommon "github.com/dsoprea/go-exif/v3/common"
)

// Metadata は抽出したExifメタデータを格納する構造体です。
type Metadata struct {
	ImageID          string    `json:"imageID"`
	FileName         string    `json:"fileName,omitempty"`
	FileSize         int64     `json:"fileSize,omitempty"`
	UploadTimestamp  time.Time `json:"uploadTimestamp,omitempty"`
	Manufacturer     string    `json:"manufacturer,omitempty"`
	Model            string    `json:"model,omitempty"`
	DateTimeOriginal time.Time `json:"dateTimeOriginal,omitempty"`
	ExposureTime     string    `json:"exposureTime,omitempty"`
	FNumber          float64   `json:"fNumber,omitempty"`
	ISOSpeedRatings  int       `json:"isoSpeedRatings,omitempty"`
	FocalLength      string    `json:"focalLength,omitempty"`
	GPSLatitude      float64   `json:"gpsLatitude,omitempty"`
	GPSLongitude     float64   `json:"gpsLongitude,omitempty"`
}

// Extract は画像データからExifメタデータを抽出します。
func Extract(r io.Reader) (*Metadata, error) {
	rawExif, err := exif.SearchAndExtractExifWithReader(r)
	if err != nil {
		if err == exif.ErrNoExif {
			return &Metadata{}, nil
		}
		return nil, err
	}

	im, err := exifcommon.NewIfdMappingWithStandard()
	if err != nil {
		return nil, err
	}

	ti := exif.NewTagIndex()
	_, index, err := exif.Collect(im, ti, rawExif)
	if err != nil {
		// EXIFデータが壊れている場合など、解析に失敗しても空のメタデータを返す
		log.Printf("[WARN] Could not collect exif data: %v", err)
		return &Metadata{}, nil
	}

	meta := &Metadata{}
	visitor := func(ifd *exif.Ifd, ite *exif.IfdTagEntry) error {
		log.Printf("[DEBUG] Found tag: Name=[%s]", ite.TagName())

		value, err := ite.Value()
		if err != nil {
			// 値がデコードできないタグはスキップします
			log.Printf("[WARN] Could not decode tag [%s]: %v", ite.TagName(), err)
			return nil
		}

		switch ite.TagName() {
		case "Make":
			meta.Manufacturer, _ = value.(string)
		case "Model":
			meta.Model, _ = value.(string)
		case "DateTimeOriginal":
			if dtStr, ok := value.(string); ok {
				meta.DateTimeOriginal, _ = time.Parse("2006:01:02 15:04:05", dtStr)
			}
		case "ExposureTime":
			if rats, ok := value.([]exifcommon.Rational); ok && len(rats) > 0 {
				meta.ExposureTime = fmt.Sprintf("%d/%d", rats[0].Numerator, rats[0].Denominator)
			}
		case "FNumber":
			if rats, ok := value.([]exifcommon.Rational); ok && len(rats) > 0 {
				meta.FNumber = float64(rats[0].Numerator) / float64(rats[0].Denominator)
			}
		case "ISOSpeedRatings":
			if isos, ok := value.([]uint16); ok && len(isos) > 0 {
				meta.ISOSpeedRatings = int(isos[0])
			}
		case "FocalLength":
			if rats, ok := value.([]exifcommon.Rational); ok && len(rats) > 0 {
				meta.FocalLength = fmt.Sprintf("%d/%d", rats[0].Numerator, rats[0].Denominator)
			}
		case "GPSLatitude":
			if gps, ok := value.(exif.GpsDegrees); ok {
				meta.GPSLatitude = gps.Decimal()
			}
		case "GPSLongitude":
			if gps, ok := value.(exif.GpsDegrees); ok {
				meta.GPSLongitude = gps.Decimal()
			}
		}
		return nil
	}

	err = index.RootIfd.EnumerateTagsRecursively(visitor)
	if err != nil {
		return nil, err
	}

	return meta, nil
}
```

### 4.3. 単体テストの実装 (`internal/exif/extractor_test.go`)

次に、`Extract`関数が正しく動作することを確認するための単体テストを記述します。テストデータの管理を容易にするため、テスト用の画像ファイルは`internal/exif/testdata`ディレクトリに配置し、テストコードからはそのファイルを読み込みます。

`internal/exif/extractor_test.go`ファイルの内容を、以下の内容に書き換えてください。

```go
package exif

import (
	os
	"path/filepath"
	testing

	"github.com/stretchr/testify/assert"
)

func openTestFile(t *testing.T, filename string) *os.File {
	t.Helper()
	path := filepath.Join("testdata", filename)
	file, err := os.Open(path)
	assert.NoError(t, err)
	return file
}

func TestExtractWithExif(t *testing.T) {
	// Open the image file.
	file := openTestFile(t, "AkihabaraKousaten.jpeg")
	defer file.Close()

	// Extract metadata.
	metadata, err := Extract(file)
	assert.NoError(t, err)
	assert.NotNil(t, metadata)

	// 必要に応じてアサーションを追加
	// assert.Equal(t, "Canon", metadata.Manufacturer)

	t.Logf("Successfully extracted: Make=[%s], Model=[%s]", metadata.Manufacturer, metadata.Model)
}

func TestExtractWithoutExif(t *testing.T) {
	// Open the image file.
	file := openTestFile(t, "no-exif.jpeg")
	defer file.Close()

	metadata, err := Extract(file)
	assert.NoError(t, err)
	assert.NotNil(t, metadata)

	// EXIF情報がないため、各フィールドはゼロ値であることを確認
	assert.Equal(t, "", metadata.Manufacturer)
	assert.Equal(t, "", metadata.Model)
	assert.True(t, metadata.DateTimeOriginal.IsZero())
	assert.Equal(t, 0.0, metadata.GPSLatitude)

	t.Logf("Successfully processed image without EXIF data.")
}
```

*(このステップを実行する前に、`internal/exif/testdata`ディレクトリにテスト用の画像ファイル `AkihabaraKousaten.jpeg` と `no-exif.jpeg` が配置されていることを確認してください。)*

### 4.4. テストの実行

作成したテストを実行して、ロジックが正しく動作することを確認します。

```bash
go test -v ./internal/exif
```

2つのテストが`PASS`となれば成功です。

## 5. 構造化ロギングの実装

システムの運用とデバッグを容易にするため、構造化ロギングを導入します。詳細は`docs/logging-design-document.md`を参照してください。

### 5.1. ロギングライブラリのインストール

`logrus`ライブラリをプロジェクトに追加します。

```bash
go get github.com/sirupsen/logrus
```

### 5.2. ロガーの初期化

`internal/exif/extractor.go`に`init`関数を追加し、ロガーの初期設定を行います。また、標準の`log`パッケージの代わりに`logrus`を使うように`import`文を修正します。

```go
// extractor.go の import 文
import (
	"fmt"
	"io"
	// "log" // 標準のlogパッケージは削除
	os
	time

	"github.com/dsoprea/go-exif/v3"
	exifcommon "github.com/dsoprea/go-exif/v3/common"
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

// ... 以降のコード
```

### 5.3. 既存ログの置き換え

`Extract`関数内の`log.Printf`を`logrus`の呼び出しに置き換えます。これにより、ログにレベルが付与され、JSON形式で出力されるようになります。

**変更前 (`log.Printf`)**
```go
// ...
	if err != nil {
		// EXIFデータが壊れている場合など、解析に失敗しても空のメタデータを返す
		log.Printf("[WARN] Could not collect exif data: %v", err)
		return &Metadata{}, nil
	}
// ...
	vizitor := func(ifd *exif.Ifd, ite *exif.IfdTagEntry) error {
		log.Printf("[DEBUG] Found tag: Name=[%s]", ite.TagName())
// ...
```

**変更後 (`logrus`)**
```go
// ...
	if err != nil {
		// EXIFデータが壊れている場合など、解析に失敗しても空のメタデータを返す
		logrus.Warnf("Could not collect exif data: %v", err)
		return &Metadata{}, nil
	}
// ...
	vizitor := func(ifd *exif.Ifd, ite *exif.IfdTagEntry) error {
		logrus.Debugf("Found tag: Name=[%s]", ite.TagName())
// ...
```

### 5.4. 動作確認

再度テストを実行し、ログの出力形式が変わったことを確認しつつ、テストが成功することを確認します。

```bash
go test -v ./internal/exif
```

## 6. Go開発 - DynamoDBリポジトリ層の実装と単体テスト

抽出したExifメタデータをDynamoDBに保存・取得するためのリポジトリ層を実装します。

### 6.1. AWS SDK for Go v2のインストール

```bash
go get github.com/aws/aws-sdk-go-v2
go get github.com/aws/aws-sdk-go-v2/config
go get github.com/aws/aws-sdk-go-v2/service/dynamodb
go get github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue
```

### 6.2. DynamoDBリポジトリ層の実装

`internal/repository/metadata_repository.go`にメタデータ保存・取得ロジックを実装します。

### 6.3. 単体テストの記述と実行

`internal/repository/metadata_repository_test.go`に単体テストを記述し、DynamoDB Localに対してテストを実行します。

## 7. Go開発 - Lambdaハンドラの実装と統合テスト

3つのLambda関数（署名付きURL発行、メタデータ抽出、メタデータ検索）のハンドラを実装し、ローカルで統合テストを行います。

### 7.1. Lambdaランタイムライブラリのインストール

```bash
go get github.com/aws/aws-lambda-go/lambda
go get github.com/aws/aws-lambda-go/events
```

### 7.2. 署名付きURL発行Lambdaハンドラの実装

`cmd/get-signed-url/main.go`にAPI Gatewayイベントをハンドリングし、S3署名付きPUT URLを生成するロジックを実装します。

### 7.3. メタデータ抽出Lambdaハンドラの実装

`cmd/extract-metadata/main.go`にS3イベントをハンドリングし、S3からの画像ダウンロード、Exif抽出、DynamoDB永続化を連携するロジックを実装します。

### 7.4. メタデータ検索Lambdaハンドラの実装

`cmd/search-metadata/main.go`にAPI Gatewayイベントをハンドリングし、DynamoDBからのメタデータ検索ロジックを実装します。

### 7.5. ローカルでの統合テスト

AWS SAM CLIを使用して、Lambdaハンドラのローカルでの動作確認を行います。

1.  **AWS SAM CLIのインストール**: 公式ドキュメントに従ってインストールします。
    *   [AWS SAM CLI のインストール](https://docs.aws.amazon.com/ja_jp/serverless-application-model/latest/developerguide/install-samcli.html)
2.  **`template.yaml`の作成**: 各Lambda関数の定義を記述します。
3.  **ローカル実行**: `sam local invoke`コマンドを使用して、各Lambda関数をローカルで実行し、動作を確認します。

## 8. AWSリソースの手動定義とIAMポリシーの指針

AWSマネジメントコンソールまたはAWS CLIを使用して、S3バケット、DynamoDBテーブル、API Gatewayなどのリソースを手動で作成します。このステップでは、IAMポリシーの最小権限の原則に基づいた具体的な記述指針も示します。

### 8.1. S3バケットの作成

*   バケット名: `go-aws-handson-image-bucket-{your-account-id}` (一意になるようにアカウントIDなどを付与)
*   パブリックアクセスブロック設定: 全てブロック
*   バージョン管理: 無効
*   イベント通知: メタデータ抽出Lambdaをトリガーするように設定（`PutObject`イベント、プレフィックス`uploads/`）

### 8.2. DynamoDBテーブルの作成

*   テーブル名: `ImageMetadata`
*   プライマリキー: `ImageID` (String)
*   グローバルセカンダリインデックス (GSI):
    *   `FileName-index` (パーティションキー: `FileName`, プロジェクション: ALL)
    *   `UploadTimestamp-index` (パーティションキー: `UploadTimestamp`, プロジェクション: ALL)
*   キャパシティモード: オンデマンド

### 8.3. IAMロールの作成とポリシーの指針

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

### 8.4. API Gatewayの作成

*   REST APIを作成し、以下のリソースとメソッドを定義します。
    *   `/signed-url` (GET): 署名付きURL発行Lambdaと統合
    *   `/metadata/{imageID}` (GET): メタデータ検索Lambdaと統合
    *   `/metadata` (GET): メタデータ検索Lambdaと統合 (クエリパラメータ`fileName`, `from`, `to`を使用)
*   デプロイステージを作成し、APIを公開します。

## 9. AWS CDK (Go) によるIaC化

手動で作成したAWSリソースを、AWS CDK (Go) を使用してIaC化します。

### 9.1. AWS CDKプロジェクトの初期化

プロジェクトルートで以下のコマンドを実行し、AWS CDKプロジェクトを初期化します。

```bash
cdk init app --language go
```

### 9.2. リソース定義のGoコード記述

`main.go`や`stack.go`などのファイルに、S3バケット、DynamoDBテーブル、Lambda関数、API Gatewayなどのリソース定義をGoコードで記述します。

### 9.3. デプロイ

```bash
cdk deploy
```

### 9.4. 手動リソースの削除

`cdk deploy`が成功したら、手動で作成したAWSリソースは削除します。

## 10. CI/CD構築 (GitHub Actions)

GitHub Actionsを使用して、Goのビルド、テスト、AWS CDKによるデプロイを自動化するCI/CDパイプラインを構築します。

### 10.1. GitHubリポジトリの作成とコードのプッシュ

GitHubに新しいリポジトリを作成し、ローカルのプロジェクトコードをプッシュします。

### 10.2. `.github/workflows/ci-cd.yml`の作成

`.github/workflows/ci-cd.yml`ファイルを作成し、以下のステップを含むワークフローを記述します。

*   Goのセットアップ
*   依存関係のインストール
*   Goのビルド
*   Goのテスト
*   AWS認証 (OIDC認証を使用)
*   AWS CDKによるデプロイ (`cdk deploy`)

### 10.3. AWS OIDC認証の設定

GitHub ActionsからAWSリソースに安全にアクセスするために、AWS OIDC認証を設定します。

### 10.4. ワークフローの実行と確認

コードをGitHubにプッシュし、GitHub Actionsワークフローが自動的に実行され、AWSリソースがデプロイされることを確認します。

## 11. AWS環境での動作確認

デプロイされたシステムがAWS環境で正しく動作することを確認します。

### 11.1. エンドツーエンドのテストシナリオ

1.  API Gateway経由で署名付きURL発行APIを呼び出し、S3署名付きPUT URLを取得します。
2.  取得した署名付きURLを使用して、S3バケットに画像をアップロードします。
3.  数秒後、メタデータ検索APIを呼び出し、アップロードした画像のExifメタデータが取得できることを確認します。

### 11.2. CloudWatch Logsでのログ確認

各Lambda関数のCloudWatch Logsグループにアクセスし、実行ログやエラーログを確認します。

### 11.3. DynamoDBコンソールでのデータ確認

DynamoDBコンソールで`ImageMetadata`テーブルにアクセスし、保存されたExifメタデータを確認します。

## 12. リソースクリーンアップ

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