package imgin

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/hexcraft-biz/her"
	_ "github.com/neofelisho/apng"
	ffmpeg "github.com/u2takey/ffmpeg-go"
	"github.com/vincent-petithory/dataurl"
	_ "golang.org/x/image/webp"
)

// ================================================================
//
// ================================================================
type Imgin struct {
	Src        string      `json:"src" form:"src" binding:"required"`
	Image      image.Image `json:"-" form:"-"`
	JpegBytes  []byte      `json:"-" form:"-"`
	DirUploads string      `json:"-" form:"-"`
}

func (i *Imgin) Validate() *her.Error {
	u, err := url.Parse(i.Src)
	if err != nil {
		return her.ErrBadRequest
	}

	switch {
	case strings.HasPrefix(u.Scheme, "http"):
		i.Src = u.String()
		if resp, err := http.Get(i.Src); err != nil {
			return her.NewError(http.StatusServiceUnavailable, err, nil)
		} else {
			if resp.StatusCode >= 400 {
				return her.ErrBadRequest
			} else if img, err := DecodeImageFromResponse(resp); err != nil {
				return her.NewError(http.StatusBadRequest, err, nil)
			} else if jpegbytes, err := EncodeToJpeg(img); err != nil {
				return her.NewError(http.StatusInternalServerError, err, nil)
			} else {
				i.Image = img
				i.JpegBytes = jpegbytes
			}
		}

	case strings.HasPrefix(u.Scheme, "data"):
		if du, err := dataurl.DecodeString(i.Src); err != nil {
			return her.NewError(http.StatusBadRequest, err, nil)
		} else if img, _, err := image.Decode(bytes.NewReader(du.Data)); err != nil {
			return her.NewError(http.StatusInternalServerError, err, nil)
		} else if jpegbytes, err := EncodeToJpeg(img); err != nil {
			return her.NewError(http.StatusInternalServerError, err, nil)
		} else {
			i.Image = img
			i.JpegBytes = jpegbytes
		}

	case i.DirUploads != "":
		if file, err := os.Open(filepath.Join(i.DirUploads, i.Src)); err != nil {
			return her.NewError(http.StatusInternalServerError, err, nil)
		} else {
			defer file.Close()
			if img, _, err := image.Decode(file); err != nil {
				return her.NewError(http.StatusInternalServerError, err, nil)
			} else if jpegbytes, err := EncodeToJpeg(img); err != nil {
				return her.NewError(http.StatusInternalServerError, err, nil)
			} else {
				i.Image = img
				i.JpegBytes = jpegbytes
			}
		}

	default:
		return her.ErrBadRequest
	}

	return nil
}

// ================================================================
//
// ================================================================
const (
	IMAGE_APNG   = "image/apng"
	IMAGE_AVIF   = "image/avif"
	IMAGE_GIF    = "image/gif"
	IMAGE_JPEG   = "image/jpeg"
	IMAGE_PNG    = "image/png"
	IMAGE_SVGXML = "image/svg+xml"
	IMAGE_WEBP   = "image/webp"
)

var ImageMIMETypes = []string{
	IMAGE_APNG,
	//IMAGE_AVIF,
	IMAGE_GIF,
	IMAGE_JPEG,
	IMAGE_PNG,
	//IMAGE_SVGXML,
	IMAGE_WEBP,
}

// ================================================================
//
// ================================================================
func DecodeImageFromResponse(resp *http.Response) (image.Image, error) {
	defer resp.Body.Close()
	img, _, err := image.Decode(resp.Body)
	return img, err
}

// ================================================================
//
// ================================================================
func EncodeToJpeg(img image.Image) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := jpeg.Encode(buf, img, nil)
	return buf.Bytes(), err
}

// ================================================================
//
// ================================================================
func JpegToDataUrl(payload []byte) string {
	return "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(payload)
}

// ================================================================
//
// ================================================================
func CropImage(img image.Image, crop image.Rectangle) (image.Image, *her.Error) {
	type subImager interface {
		SubImage(r image.Rectangle) image.Image
	}

	// img is an Image interface. This checks if the underlying value has a
	// method called SubImage. If it does, then we can use SubImage to crop the
	// image.
	simg, ok := img.(subImager)
	if !ok {
		return nil, her.NewErrorWithMessage(http.StatusInternalServerError, "Image cropping failed", nil)
	}

	return simg.SubImage(crop), nil
}

// ================================================================
//
// ================================================================
func Mp4ToImages(sour, destDir string, fps float32) error {
	return ffmpeg.Input(sour).Output(
		filepath.Join(destDir, "%04d.jpeg"),
		ffmpeg.KwArgs{
			"v":      "error",
			"vf":     fmt.Sprintf("fps=%.2f", fps),
			"format": "image2",
			"vcodec": "mjpeg",
		},
	).Run()
}
