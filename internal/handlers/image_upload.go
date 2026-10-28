package handlers

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

// MaxImageSize defines the 20 MB upper limit for uploaded post images.
const MaxImageSize = 20 << 20

var (
	errImageTooLarge    = errors.New("The image is too big. Maximum allowed size is 20 MB.")
	errInvalidImageType = errors.New("Unsupported image format. Allowed formats: JPEG, PNG, GIF, SVG.")
)

type uploadedImage struct {
	Data     []byte
	MIMEType string
	Ext      string
}

// parsePostFormAndImage parses either a multipart/form-data or application/x-www-form-urlencoded request
// and validates any attached image (max 20 MB; JPEG, PNG, GIF, or SVG).
func parsePostFormAndImage(r *http.Request) (*uploadedImage, error) {
	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		if err := r.ParseMultipartForm(MaxImageSize + (1 << 20)); err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "request body too large") {
				return nil, errImageTooLarge
			}
			return nil, errors.New("Invalid multipart form data.")
		}
	} else {
		if err := r.ParseForm(); err != nil {
			return nil, errors.New("Invalid form data.")
		}
		return nil, nil
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			return nil, nil
		}
		return nil, errors.New("Unable to read uploaded image.")
	}
	defer file.Close()

	if header.Size > MaxImageSize {
		return nil, errImageTooLarge
	}

	data, err := io.ReadAll(io.LimitReader(file, MaxImageSize+1))
	if err != nil {
		return nil, errors.New("Failed to read uploaded image.")
	}
	if int64(len(data)) > MaxImageSize {
		return nil, errImageTooLarge
	}
	if len(data) == 0 {
		return nil, nil
	}

	mimeType, ext, ok := detectAllowedImage(data, header.Filename)
	if !ok {
		return nil, errInvalidImageType
	}

	return &uploadedImage{
		Data:     data,
		MIMEType: mimeType,
		Ext:      ext,
	}, nil
}

// detectAllowedImage verifies both file magic bytes and extension for JPEG, PNG, GIF, and SVG.
func detectAllowedImage(data []byte, filename string) (string, string, bool) {
	ext := strings.ToLower(filepath.Ext(filename))
	sniff := http.DetectContentType(data)

	switch {
	case sniff == "image/jpeg" && (ext == ".jpg" || ext == ".jpeg" || ext == ""):
		return "image/jpeg", ".jpg", true
	case sniff == "image/png" && (ext == ".png" || ext == ""):
		return "image/png", ".png", true
	case sniff == "image/gif" && (ext == ".gif" || ext == ""):
		return "image/gif", ".gif", true
	case ext == ".svg" && isValidSVG(data):
		return "image/svg+xml", ".svg", true
	default:
		return "", "", false
	}
}

func isValidSVG(data []byte) bool {
	sample := data
	if len(sample) > 4096 {
		sample = sample[:4096]
	}
	trimmed := bytes.TrimSpace(sample)
	lower := bytes.ToLower(trimmed)
	if bytes.Contains(lower, []byte("<script")) {
		return false
	}
	return bytes.Contains(lower, []byte("<svg"))
}

// saveUploadedImage writes the validated image to uploads/<uuid><ext> and returns its URL path.
func saveUploadedImage(img *uploadedImage) (string, error) {
	if err := os.MkdirAll("uploads", 0755); err != nil {
		return "", err
	}
	id, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	filename := id.String() + img.Ext
	diskPath := filepath.Join("uploads", filename)
	if err := os.WriteFile(diskPath, img.Data, 0644); err != nil {
		return "", err
	}
	return "/uploads/" + filename, nil
}

// UploadsHandler serves uploaded post images from the uploads/ directory with safe headers.
func UploadsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		ErrorPageHandler(w, r, http.StatusMethodNotAllowed, errMethodNotAllowedTitle, errMethodNotAllowedMessage)
		return
	}

	rel := strings.TrimPrefix(r.URL.Path, "/uploads/")
	if rel == "" || strings.Contains(rel, "/") || strings.Contains(rel, "\\") || strings.Contains(rel, "..") {
		ErrorPageHandler(w, r, http.StatusNotFound, "404 Not Found", "Image not found.")
		return
	}

	diskPath := filepath.Join("uploads", filepath.Clean(rel))
	info, err := os.Stat(diskPath)
	if err != nil || info.IsDir() {
		ErrorPageHandler(w, r, http.StatusNotFound, "404 Not Found", "Image not found.")
		return
	}

	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'")
	http.ServeFile(w, r, diskPath)
}
