package exif

import (
	"fmt"
	"io"
	"os"
	"time"

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

// Metadata は抽出したExifメタデータを格納する構造体です。
type Metadata struct {
	ImageID          string    `json:"imageID"`
	FileName         string    `json:"fileName"`
	FileSize         int64     `json:"fileSize"`
	UploadTimestamp  time.Time `json:"uploadTimestamp"`
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
		logrus.Warnf("Could not collect exif data: %v", err)
		return &Metadata{}, nil
	}

	meta := &Metadata{}
	visitor := func(ifd *exif.Ifd, ite *exif.IfdTagEntry) error {
		logrus.Debugf("Found tag: Name=[%s]", ite.TagName())

		value, err := ite.Value()
		if err != nil {
			// 値がデコードできないタグはスキップします
			logrus.Warnf("Could not decode tag [%s]: %v", ite.TagName(), err)
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
		}
		return nil
	}

	err = index.RootIfd.EnumerateTagsRecursively(visitor)
	if err != nil {
		return nil, err
	}

	// GPS IFDからGPS情報を取得
	if gpsIfd, err := exif.FindIfdFromRootIfd(index.RootIfd, "IFD/GPSInfo"); err == nil {
		if gpsInfo, err := gpsIfd.GpsInfo(); err == nil {
			meta.GPSLatitude = gpsInfo.Latitude.Decimal()
			meta.GPSLongitude = gpsInfo.Longitude.Decimal()
		} else {
			logrus.Warnf("Could not parse GPS info from GPS IFD: %v", err)
		}
	} else if err.Error() != "tag not found" {
		// ErrTagNotFoundは「GPSタグがない」という正常なケースなのでログは不要。
		// それ以外の予期せぬエラーの場合のみログを出力する。
		logrus.Warnf("Could not find GPS IFD: %v", err)
	}

	return meta, nil
}