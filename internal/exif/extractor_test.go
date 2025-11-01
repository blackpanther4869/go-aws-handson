package exif

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// openTestFile はテストデータファイルを開くヘルパー関数です。
func openTestFile(t *testing.T, filename string) *os.File {
	t.Helper()
	path := filepath.Join("testdata", filename)
	file, err := os.Open(path)
	assert.NoError(t, err)
	return file
}

// loadExpectation は期待結果のJSONファイルを読み込み、Metadata構造体にデコードします。
func loadExpectation(t *testing.T, filename string) *Metadata {
	t.Helper()
	path := filepath.Join("testdata", filename)
	data, err := os.ReadFile(path)
	assert.NoError(t, err)

	var expectation Metadata
	err = json.Unmarshal(data, &expectation)
	assert.NoError(t, err)

	return &expectation
}

func TestExtract(t *testing.T) {
	testCases := []struct {
		name         string
		inputFile    string
		expectedFile string
		expectError  bool
	}{
		{
			name:         "正常系: EXIF情報を持つJPEG (Fujifilm)",
			inputFile:    "Fujifilm_FinePix_E500.jpg",
			expectedFile: "fujifilm-finepix-e500.json",
			expectError:  false,
		},
		{
			name:         "正常系: GPS情報を持つJPEG",
			inputFile:    "gps.jpg",
			expectedFile: "gps.json",
			expectError:  false,
		},
		{
			name:         "正常系: EXIF情報を持つJPEG (Canon)",
			inputFile:    "canon-ixus.jpg",
			expectedFile: "canon-ixus.json",
			expectError:  false,
		},
		{
			name:         "正常系: EXIF情報を持たないJPEG (from samples)",
			inputFile:    "no-exif.jpg",
			expectedFile: "without-exif.json",
			expectError:  false,
		},
		{
			name:         "異常系: 破損したJPEG",
			inputFile:    "corrupt.jpg",
			expectedFile: "corrupt.json",
			expectError:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			file := openTestFile(t, tc.inputFile)
			defer file.Close()
			expectation := loadExpectation(t, tc.expectedFile)

			// Act
			actual, err := Extract(file)

			// Assert
			if tc.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, actual)

			// Extract関数の責務外のフィールドは比較対象から除外
			expectation.ImageID = actual.ImageID
			expectation.FileName = actual.FileName
			expectation.FileSize = actual.FileSize
			expectation.UploadTimestamp = actual.UploadTimestamp

			assert.Equal(t, expectation, actual)
		})
	}
}