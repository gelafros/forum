package handlers

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

func buildMultipartPostRequest(t *testing.T, title, content, categoryID, filename string, fileBytes []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	_ = writer.WriteField("title", title)
	_ = writer.WriteField("content", content)
	_ = writer.WriteField("categories", categoryID)

	if filename != "" {
		part, err := writer.CreateFormFile("image", filename)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write(fileBytes); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/create_post", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func TestCreatePostWithImagesAndSizeLimit(t *testing.T) {
	testDB := setupHandlerTestDB(t)
	t.Cleanup(func() { _ = os.RemoveAll("uploads") })

	userID := addUser(t, testDB, "imguser", "img@example.com", "secret")
	if _, err := testDB.Exec("INSERT INTO sessions (user_id, token, expires_at) VALUES (?, ?, ?)", userID, "img-token", time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	sessionCookie := &http.Cookie{Name: "session_token", Value: "img-token"}

	// Valid minimal PNG, GIF, JPEG, and SVG payloads
	pngBytes := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0x00, 0x00, 0x00, 0x0d, 'I', 'H', 'D', 'R'}
	gifBytes := []byte("GIF89a\x01\x00\x01\x00\x80\x00\x00\xff\xff\xff\x00\x00\x00!\xf9\x04\x01\x00\x00\x00\x00,\x00\x00\x00\x00\x01\x00\x01\x00\x00\x02\x02D\x01\x00;")
	jpgBytes := []byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00, 0x01, 0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0xff, 0xd9}
	svgBytes := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="10" height="10"><rect width="10" height="10"/></svg>`)

	formats := []struct {
		name string
		file string
		data []byte
	}{
		{"PNG", "photo.png", pngBytes},
		{"GIF", "anim.gif", gifBytes},
		{"JPEG", "pic.jpg", jpgBytes},
		{"SVG", "vector.svg", svgBytes},
	}

	for _, tc := range formats {
		t.Run("Valid_"+tc.name, func(t *testing.T) {
			req := buildMultipartPostRequest(t, "Post "+tc.name, "Body with image", "1", tc.file, tc.data)
			req.AddCookie(sessionCookie)
			rec := httptest.NewRecorder()
			CreatePostHandler(rec, req)
			if rec.Code != http.StatusSeeOther {
				t.Fatalf("expected 303 redirect for %s upload, got %d: %s", tc.name, rec.Code, rec.Body.String())
			}
		})
	}

	// Verify that a guest viewing /post?id=1 sees the associated uploaded image
	var imagePath string
	if err := testDB.QueryRow("SELECT image_path FROM post_images WHERE post_id = 1").Scan(&imagePath); err != nil || imagePath == "" {
		t.Fatalf("expected stored image_path for post 1, got %q (err=%v)", imagePath, err)
	}

	guestReq := httptest.NewRequest(http.MethodGet, "/post?id=1", nil)
	guestRec := httptest.NewRecorder()
	PostDetailHandler(guestRec, guestReq)
	if guestRec.Code != http.StatusOK || !strings.Contains(guestRec.Body.String(), imagePath) {
		t.Fatalf("expected guest view to include image %q, status=%d", imagePath, guestRec.Code)
	}

	// Verify that /uploads/ serves the uploaded image
	uploadReq := httptest.NewRequest(http.MethodGet, imagePath, nil)
	uploadRec := httptest.NewRecorder()
	UploadsHandler(uploadRec, uploadReq)
	if uploadRec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from UploadsHandler for %s, got %d", imagePath, uploadRec.Code)
	}

	// Reject image larger than 20 MB with clear message
	oversized := make([]byte, MaxImageSize+1024)
	copy(oversized, pngBytes)
	bigReq := buildMultipartPostRequest(t, "Too Big", "Oversized image", "1", "huge.png", oversized)
	bigReq.AddCookie(sessionCookie)
	bigRec := httptest.NewRecorder()
	CreatePostHandler(bigRec, bigReq)
	if bigRec.Code != http.StatusBadRequest || !strings.Contains(bigRec.Body.String(), "20 MB") {
		t.Fatalf("expected 400 Bad Request with 20 MB error message, got status=%d", bigRec.Code)
	}

	// Reject unsupported file format
	badReq := buildMultipartPostRequest(t, "Bad File", "Text file", "1", "notes.txt", []byte("plain text not an image"))
	badReq.AddCookie(sessionCookie)
	badRec := httptest.NewRecorder()
	CreatePostHandler(badRec, badReq)
	if badRec.Code != http.StatusBadRequest || !strings.Contains(badRec.Body.String(), "Unsupported image format") {
		t.Fatalf("expected 400 Bad Request for unsupported format, got status=%d", badRec.Code)
	}
	_ = strconv.Itoa(userID)
}
