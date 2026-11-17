package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestModerationRolesReportsApprovalAndCategories(t *testing.T) {
	testDB := setupHandlerTestDB(t)

	userID := addUser(t, testDB, "normaluser", "user@example.com", "secret")
	adminID := addUser(t, testDB, "adminuser", "admin@example.com", "secret")

	if _, err := testDB.Exec("INSERT INTO user_roles (user_id, role) VALUES (?, 'admin')", adminID); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB.Exec("INSERT INTO sessions (user_id, token, expires_at) VALUES (?, 'user-tok', ?), (?, 'admin-tok', ?)",
		userID, time.Now().Add(time.Hour), adminID, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	userCookie := &http.Cookie{Name: "session_token", Value: "user-tok"}
	adminCookie := &http.Cookie{Name: "session_token", Value: "admin-tok"}

	// 1. User requests moderator role
	reqRole := httptest.NewRequest(http.MethodPost, "/moderation/request-role", nil)
	reqRole.AddCookie(userCookie)
	recRole := httptest.NewRecorder()
	RequestModeratorRoleHandler(recRole, reqRole)
	if recRole.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 from RequestModeratorRoleHandler, got %d", recRole.Code)
	}

	// 2. Admin promotes user to moderator
	promoteForm := url.Values{"user_id": {strconv.Itoa(userID)}, "role": {"moderator"}}
	promoteReq := httptest.NewRequest(http.MethodPost, "/moderation/role", strings.NewReader(promoteForm.Encode()))
	promoteReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	promoteReq.AddCookie(adminCookie)
	promoteRec := httptest.NewRecorder()
	ManageRoleHandler(promoteRec, promoteReq)
	if promoteRec.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 from ManageRoleHandler, got %d", promoteRec.Code)
	}

	// 3. Create a pending post under MODERATION_PREAPPROVAL=true by a third normal user
	authorID := addUser(t, testDB, "poster", "poster@example.com", "secret")
	if _, err := testDB.Exec("INSERT INTO sessions (user_id, token, expires_at) VALUES (?, 'poster-tok', ?)", authorID, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MODERATION_PREAPPROVAL", "true")
	postForm := url.Values{"title": {"Moderated Post"}, "content": {"Needs approval"}, "categories": {"1"}}
	postReq := httptest.NewRequest(http.MethodPost, "/create_post", strings.NewReader(postForm.Encode()))
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postReq.AddCookie(&http.Cookie{Name: "session_token", Value: "poster-tok"})
	postRec := httptest.NewRecorder()
	CreatePostHandler(postRec, postReq)
	if postRec.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 creating pending post, got %d", postRec.Code)
	}

	// Guest should NOT see the pending post on homepage yet
	guestHome := httptest.NewRequest(http.MethodGet, "/", nil)
	guestRec := httptest.NewRecorder()
	HomeHandler(guestRec, guestHome)
	if strings.Contains(guestRec.Body.String(), "Moderated Post") {
		t.Fatal("pending post must not be visible to guests before moderator approval")
	}

	// Guest must also get 404 opening the pending post by direct URL
	guestDetailRec := httptest.NewRecorder()
	PostDetailHandler(guestDetailRec, httptest.NewRequest(http.MethodGet, "/post?id=1", nil))
	if guestDetailRec.Code != http.StatusNotFound {
		t.Fatalf("pending post by direct URL: guest status = %d, want 404", guestDetailRec.Code)
	}

	// Author can still view their own pending post
	authorDetailReq := httptest.NewRequest(http.MethodGet, "/post?id=1", nil)
	authorDetailReq.AddCookie(&http.Cookie{Name: "session_token", Value: "poster-tok"})
	authorDetailRec := httptest.NewRecorder()
	PostDetailHandler(authorDetailRec, authorDetailReq)
	if authorDetailRec.Code != http.StatusOK {
		t.Fatalf("pending post by direct URL: author status = %d, want 200", authorDetailRec.Code)
	}

	// 4. Moderator approves the pending post
	approveForm := url.Values{"post_id": {"1"}}
	approveReq := httptest.NewRequest(http.MethodPost, "/moderation/post/approve", strings.NewReader(approveForm.Encode()))
	approveReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	approveReq.AddCookie(userCookie)
	approveRec := httptest.NewRecorder()
	ApprovePostHandler(approveRec, approveReq)
	if approveRec.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 from ApprovePostHandler, got %d", approveRec.Code)
	}

	// Guest now sees the approved post
	guestRec2 := httptest.NewRecorder()
	HomeHandler(guestRec2, httptest.NewRequest(http.MethodGet, "/", nil))
	if !strings.Contains(guestRec2.Body.String(), "Moderated Post") {
		t.Fatal("approved post should be visible to guests")
	}

	// 5. Moderator reports the post to Admin as "obscene", and Admin responds + deletes post
	repForm := url.Values{"post_id": {"1"}, "reason": {"obscene"}, "details": {"Inappropriate language"}}
	repReq := httptest.NewRequest(http.MethodPost, "/moderation/report", strings.NewReader(repForm.Encode()))
	repReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	repReq.AddCookie(userCookie)
	repRec := httptest.NewRecorder()
	ReportPostHandler(repRec, repReq)
	if repRec.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 from ReportPostHandler, got %d", repRec.Code)
	}

	respForm := url.Values{"report_id": {"1"}, "status": {"resolved"}, "admin_response": {"Removed post"}, "delete_post": {"true"}}
	respReq := httptest.NewRequest(http.MethodPost, "/moderation/report/respond", strings.NewReader(respForm.Encode()))
	respReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	respReq.AddCookie(adminCookie)
	respRec := httptest.NewRecorder()
	RespondReportHandler(respRec, respReq)
	if respRec.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 from RespondReportHandler, got %d", respRec.Code)
	}

	// Verify the resolved report is preserved as an audit record even after the post is deleted
	var repStatus, repAdminResp string
	if err := testDB.QueryRow("SELECT status, admin_response FROM moderation_reports WHERE id = 1").Scan(&repStatus, &repAdminResp); err != nil {
		t.Fatalf("expected resolved moderation report to remain after post deletion: %v", err)
	}
	if repStatus != "resolved" || repAdminResp != "Removed post" {
		t.Fatalf("unexpected report state: status=%q admin_response=%q", repStatus, repAdminResp)
	}

	// 6. Admin creates and deletes a category
	catForm := url.Values{"action": {"create"}, "name": {"Science"}}
	catReq := httptest.NewRequest(http.MethodPost, "/moderation/categories", strings.NewReader(catForm.Encode()))
	catReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	catReq.AddCookie(adminCookie)
	catRec := httptest.NewRecorder()
	ManageCategoriesHandler(catRec, catReq)
	if catRec.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 creating category, got %d", catRec.Code)
	}
}
