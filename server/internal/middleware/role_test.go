package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/backup-saas/server/internal/models"
	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// authMatrix defines the expected HTTP status for each role on a given endpoint.
type authMatrix struct {
	name     string
	roles    []models.MemberRole // roles that are ALLOWED
	endpoint string
}

// requireRoleResult calls RequireRole with the given allowed roles and returns
// the status code for a request made by a member with memberRole.
func requireRoleResult(allowedRoles []models.MemberRole, memberRole models.MemberRole) int {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("member_role", string(memberRole))

	handler := RequireRole(allowedRoles...)
	handler(c)

	if w.Code == 0 {
		return http.StatusOK // middleware called Next(), handler would run
	}
	return w.Code
}

func TestRequireRole_Matrix(t *testing.T) {
	allRoles := []models.MemberRole{
		models.RoleOwner,
		models.RoleAdmin,
		models.RoleOperator,
		models.RoleViewer,
	}

	tests := []struct {
		endpoint     string
		allowedRoles []models.MemberRole
		wantAllow    []models.MemberRole
		wantDeny     []models.MemberRole
	}{
		// Read-only endpoints — all roles allowed
		{
			endpoint:     "GET /backup-jobs",
			allowedRoles: allRoles,
			wantAllow:    []models.MemberRole{models.RoleOwner, models.RoleAdmin, models.RoleOperator, models.RoleViewer},
			wantDeny:     nil,
		},
		{
			endpoint:     "GET /backup-runs",
			allowedRoles: allRoles,
			wantAllow:    []models.MemberRole{models.RoleOwner, models.RoleAdmin, models.RoleOperator, models.RoleViewer},
			wantDeny:     nil,
		},
		// Create/run job — OWNER, ADMIN, OPERATOR
		{
			endpoint:     "POST /backup-jobs",
			allowedRoles: []models.MemberRole{models.RoleOwner, models.RoleAdmin, models.RoleOperator},
			wantAllow:    []models.MemberRole{models.RoleOwner, models.RoleAdmin, models.RoleOperator},
			wantDeny:     []models.MemberRole{models.RoleViewer},
		},
		{
			endpoint:     "POST /backup-jobs/:id/run",
			allowedRoles: []models.MemberRole{models.RoleOwner, models.RoleAdmin, models.RoleOperator},
			wantAllow:    []models.MemberRole{models.RoleOwner, models.RoleAdmin, models.RoleOperator},
			wantDeny:     []models.MemberRole{models.RoleViewer},
		},
		{
			endpoint:     "POST /backup-runs/:id/cancel",
			allowedRoles: []models.MemberRole{models.RoleOwner, models.RoleAdmin, models.RoleOperator},
			wantAllow:    []models.MemberRole{models.RoleOwner, models.RoleAdmin, models.RoleOperator},
			wantDeny:     []models.MemberRole{models.RoleViewer},
		},
		{
			endpoint:     "POST /restores",
			allowedRoles: []models.MemberRole{models.RoleOwner, models.RoleAdmin, models.RoleOperator},
			wantAllow:    []models.MemberRole{models.RoleOwner, models.RoleAdmin, models.RoleOperator},
			wantDeny:     []models.MemberRole{models.RoleViewer},
		},
		// Delete job — OWNER, ADMIN only
		{
			endpoint:     "DELETE /backup-jobs/:id",
			allowedRoles: []models.MemberRole{models.RoleOwner, models.RoleAdmin},
			wantAllow:    []models.MemberRole{models.RoleOwner, models.RoleAdmin},
			wantDeny:     []models.MemberRole{models.RoleOperator, models.RoleViewer},
		},
		// Enable/disable job — OWNER, ADMIN only
		{
			endpoint:     "POST /backup-jobs/:id/enable",
			allowedRoles: []models.MemberRole{models.RoleOwner, models.RoleAdmin},
			wantAllow:    []models.MemberRole{models.RoleOwner, models.RoleAdmin},
			wantDeny:     []models.MemberRole{models.RoleOperator, models.RoleViewer},
		},
		// Storage management — OWNER, ADMIN only
		{
			endpoint:     "POST /storage-targets",
			allowedRoles: []models.MemberRole{models.RoleOwner, models.RoleAdmin},
			wantAllow:    []models.MemberRole{models.RoleOwner, models.RoleAdmin},
			wantDeny:     []models.MemberRole{models.RoleOperator, models.RoleViewer},
		},
		{
			endpoint:     "DELETE /storage-targets/:id",
			allowedRoles: []models.MemberRole{models.RoleOwner, models.RoleAdmin},
			wantAllow:    []models.MemberRole{models.RoleOwner, models.RoleAdmin},
			wantDeny:     []models.MemberRole{models.RoleOperator, models.RoleViewer},
		},
		// Member management — OWNER, ADMIN only
		{
			endpoint:     "POST /members",
			allowedRoles: []models.MemberRole{models.RoleOwner, models.RoleAdmin},
			wantAllow:    []models.MemberRole{models.RoleOwner, models.RoleAdmin},
			wantDeny:     []models.MemberRole{models.RoleOperator, models.RoleViewer},
		},
		{
			endpoint:     "PUT /members/:user_id/role",
			allowedRoles: []models.MemberRole{models.RoleOwner, models.RoleAdmin},
			wantAllow:    []models.MemberRole{models.RoleOwner, models.RoleAdmin},
			wantDeny:     []models.MemberRole{models.RoleOperator, models.RoleViewer},
		},
		{
			endpoint:     "DELETE /members/:user_id",
			allowedRoles: []models.MemberRole{models.RoleOwner, models.RoleAdmin},
			wantAllow:    []models.MemberRole{models.RoleOwner, models.RoleAdmin},
			wantDeny:     []models.MemberRole{models.RoleOperator, models.RoleViewer},
		},
		// Agent token management — OWNER, ADMIN only
		{
			endpoint:     "POST /agents/token",
			allowedRoles: []models.MemberRole{models.RoleOwner, models.RoleAdmin},
			wantAllow:    []models.MemberRole{models.RoleOwner, models.RoleAdmin},
			wantDeny:     []models.MemberRole{models.RoleOperator, models.RoleViewer},
		},
		{
			endpoint:     "POST /agents/:id/rotate-token",
			allowedRoles: []models.MemberRole{models.RoleOwner, models.RoleAdmin},
			wantAllow:    []models.MemberRole{models.RoleOwner, models.RoleAdmin},
			wantDeny:     []models.MemberRole{models.RoleOperator, models.RoleViewer},
		},
		{
			endpoint:     "DELETE /agents/:id",
			allowedRoles: []models.MemberRole{models.RoleOwner, models.RoleAdmin},
			wantAllow:    []models.MemberRole{models.RoleOwner, models.RoleAdmin},
			wantDeny:     []models.MemberRole{models.RoleOperator, models.RoleViewer},
		},
	}

	for _, tt := range tests {
		t.Run(tt.endpoint, func(t *testing.T) {
			for _, role := range tt.wantAllow {
				code := requireRoleResult(tt.allowedRoles, role)
				// Middleware called Next() → recorder stays at 0 (no response written)
				if code != 0 && code != http.StatusOK {
					t.Errorf("%s: role %s should be ALLOWED, got %d", tt.endpoint, role, code)
				}
			}
			for _, role := range tt.wantDeny {
				code := requireRoleResult(tt.allowedRoles, role)
				if code != http.StatusForbidden {
					t.Errorf("%s: role %s should be DENIED (403), got %d", tt.endpoint, role, code)
				}
			}
		})
	}
}

// TestRequireRole_MissingRole verifies that a request with no role set is denied.
func TestRequireRole_MissingRole(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	// member_role not set in context

	handler := RequireRole(models.RoleOwner)
	handler(c)

	if w.Code != http.StatusForbidden {
		t.Errorf("missing role should be denied, got %d", w.Code)
	}
}

// TestRequireRole_UnknownRole verifies that an unrecognized role string is denied.
func TestRequireRole_UnknownRole(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("member_role", "SUPERUSER") // not a real role

	handler := RequireRole(models.RoleOwner, models.RoleAdmin)
	handler(c)

	if w.Code != http.StatusForbidden {
		t.Errorf("unknown role should be denied, got %d", w.Code)
	}
}
