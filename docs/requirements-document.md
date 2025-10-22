## 🚀 AWS構築＆Go開発ハンズオン - 機能要件定義書

### 1. 画像アップロード機能

*   **クライアントからS3署名付きURLの取得**:
    *   クライアントは、API Gateway経由で公開されるLambda関数（署名付きURL発行Lambda）を呼び出し、アップロード対象のファイル名とファイルタイプ（MIME Type）を渡します。
    *   Lambda関数は、指定されたファイル名とファイルタイプに対応するS3署名付きPUT URLを生成し、クライアントに返却します。
    *   署名付きURLの有効期限は、セキュリティと利便性を考慮し、**5分間**とします。
*   **アップロード可能な画像の種類と制約**:
    *   アップロード可能な画像形式は、**JPEG (.jpg, .jpeg)** および **TIFF (.tif, .tiff)** とします。
    *   ファイルサイズは、Lambdaのメモリや処理時間を考慮し、**最大10MB**とします。
    *   S3バケットは、アップロードされた画像を格納する専用のバケットとします。

### 2. Exifメタデータ抽出機能

*   **S3イベントトリガーの設定**:
    *   画像アップロード用S3バケットの`PutObject`イベントをトリガーとして、メタデータ抽出Lambda関数を起動します。
    *   特定のプレフィックス（例: `uploads/`）配下にアップロードされたオブジェクトのみを対象とします。
*   **抽出対象のExifメタデータ項目**:
    *   以下のExifメタデータ項目を抽出対象とします。これらはDynamoDBに永続化されます。
        *   `ImageID` (DynamoDBのプライマリキーとしてLambdaで生成)
        *   `FileName` (S3オブジェクトキー)
        *   `FileSize` (S3オブジェクトサイズ)
        *   `UploadTimestamp` (S3オブジェクトの最終更新日時)
        *   `Make` (カメラメーカー)
        *   `Model` (カメラモデル)
        *   `DateTimeOriginal` (撮影日時)
        *   `ExposureTime` (露出時間)
        *   `FNumber` (F値)
        *   `ISOSpeedRatings` (ISO感度)
        *   `FocalLength` (焦点距離)
        *   `GPSLatitude` (GPS緯度)
        *   `GPSLongitude` (GPS経度)
        *   `GPSAltitude` (GPS高度)
        *   `Orientation` (画像の向き)
    *   上記以外のExifデータは、必要に応じて追加検討します。
*   **エラーハンドリング**:
    *   アップロードされたファイルが画像ファイルではない場合、またはExifデータが含まれていない場合は、エラーログを出力し、処理をスキップします。
    *   Exifデータの抽出に失敗した場合も、エラーログを出力し、処理をスキップします。

### 3. 抽出したメタデータのDynamoDB永続化機能

*   **DynamoDBテーブル構造**:
    *   テーブル名: `ImageMetadata`
    *   プライマリキー: `ImageID` (String, UUIDなどで一意に生成)
    *   ソートキー: なし（シンプルなキー設計）
    *   グローバルセカンダリインデックス (GSI):
        *   `FileName-index` (パーティションキー: `FileName`, プロジェクション: ALL) - ファイル名での検索用
        *   `UploadTimestamp-index` (パーティションキー: `UploadTimestamp`, プロジェクション: ALL) - アップロード日時での検索用
*   **保存するメタデータの具体的な項目とデータ型**:
    *   上記「抽出対象のExifメタデータ項目」で挙げた項目をDynamoDBの属性として保存します。
    *   データ型は、DynamoDBのデータ型（String, Numberなど）にマッピングします。
        *   `ImageID`: String
        *   `FileName`: String
        *   `FileSize`: Number
        *   `UploadTimestamp`: String (ISO 8601形式)
        *   `Make`: String
        *   `Model`: String
        *   `DateTimeOriginal`: String (ISO 8601形式)
        *   `ExposureTime`: String
        *   `FNumber`: String
        *   `ISOSpeedRatings`: Number
        *   `FocalLength`: String
        *   `GPSLatitude`: Number
        *   `GPSLongitude`: Number
        *   `GPSAltitude`: Number
        *   `Orientation`: Number
*   **更新時の挙動**:
    *   同じ`FileName`の画像が再度アップロードされた場合、既存のメタデータを**上書き**します。`ImageID`は新規に生成されます。

### 4. メタデータ検索/取得機能

*   **APIインターフェース**:
    *   API Gateway経由で公開されるLambda関数（メタデータ検索Lambda）を呼び出します。
    *   **GET /metadata/{imageID}**: 特定の`ImageID`を持つメタデータを取得します。
    *   **GET /metadata?fileName={fileName}**: 特定の`FileName`を持つメタデータを取得します。
    *   **GET /metadata?from={timestamp}&to={timestamp}**: 指定された期間内にアップロードされたメタデータを取得します。
    *   レスポンス形式はJSONとします。
*   **検索条件**:
    *   `ImageID`による直接取得。
    *   `FileName`による検索（GSI `FileName-index`を使用）。
    *   `UploadTimestamp`の範囲指定による検索（GSI `UploadTimestamp-index`を使用）。
    *   将来的には、特定のExifタグの値（例: `Make=Canon`）での検索も検討しますが、まずは上記3つの検索条件を実装します。