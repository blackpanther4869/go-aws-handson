# API Gateway 設計書

## 1. 概要

本システムへのリクエストを受け付けるエンドポイントとして、Amazon API Gatewayを利用する。
クライアントからのリクエストを検証し、適切なLambda関数へルーティングする責務を持つ。

## 2. APIエンドポイント定義

| エンドポイント | メソッド | 説明 | 統合先Lambda |
|:---|:---|:---|:---|
| `/signed-url` | GET | 画像アップロード用の署名付きURLを発行する。 | 署名付きURL発行Lambda |
| `/metadata/{imageID}` | GET | 指定された `imageID` のメタデータを取得する。 | メタデータ検索Lambda |
| `/metadata` | GET | クエリパラメータに基づきメタデータを検索する。 | メタデータ検索Lambda |

## 3. リクエスト/レスポンス仕様

### 3.1. `GET /signed-url`

- **クエリパラメータ**:
    - `fileName` (string, 必須): アップロードするファイル名。
    - `contentType` (string, 必須): ファイルのMIMEタイプ (例: `image/jpeg`)。
- **成功レスポンス (200 OK)**:
    ```json
    {
        "uploadUrl": "https://...",
        "imageUrl": "https://..."
    }
    ```
- **エラーレスポンス (400 Bad Request)**:
    ```json
    {
        "message": "fileName and contentType are required."
    }
    ```

### 3.2. `GET /metadata/{imageID}`

- **パスパラメータ**:
    - `imageID` (string, 必須): 検索対象の画像ID。
- **成功レスポンス (200 OK)**:
    - `architecture-design-document.md` のデータモデルに準拠したJSONオブジェクト。
- **エラーレスポンス (404 Not Found)**:
    ```json
    {
        "message": "Metadata not found."
    }
    ```

### 3.3. `GET /metadata`

- **クエリパラメータ**:
    - `fileName` (string): ファイル名による検索。
    - `from` (string): アップロード日時の範囲検索（開始、ISO 8601形式）。
    - `to` (string): アップロード日時の範囲検索（終了、ISO 8601形式）。
- **成功レスポンス (200 OK)**:
    - `architecture-design-document.md` のデータモデルに準拠したJSONオブジェクトの配列。
- **エラーレスポンス (400 Bad Request)**:
    ```json
    {
        "message": "Invalid query parameter."
    }
    ```

## 4. 認証・認可

- 本ハンズオンでは、APIキーやIAM認証などの認証メカニズムは導入しない。
- 全てのエンドポイントは公開状態とする。

## 5. その他

- デプロイステージは `dev` とする。
- カスタムドメインは設定しない。