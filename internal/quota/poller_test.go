package quota

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/router-for-me/CLIProxyAPIBusiness/internal/db"
	"github.com/router-for-me/CLIProxyAPIBusiness/internal/models"
	"gorm.io/datatypes"
)

func TestMarkAuthUnavailableClearsQuota(t *testing.T) {
	conn, errOpen := db.Open(":memory:")
	if errOpen != nil {
		t.Fatalf("open db: %v", errOpen)
	}
	if errMigrate := db.Migrate(conn); errMigrate != nil {
		t.Fatalf("migrate db: %v", errMigrate)
	}

	now := time.Now().UTC()
	auth := models.Auth{
		Key:         "auth-1",
		Content:     datatypes.JSON([]byte(`{"type":"codex"}`)),
		IsAvailable: true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if errCreate := conn.Create(&auth).Error; errCreate != nil {
		t.Fatalf("create auth: %v", errCreate)
	}
	quota := models.Quota{
		AuthID:    auth.ID,
		Type:      "codex",
		Data:      datatypes.JSON([]byte(`{"usage":"stale"}`)),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if errCreate := conn.Create(&quota).Error; errCreate != nil {
		t.Fatalf("create quota: %v", errCreate)
	}

	poller := &Poller{db: conn}
	row := authRowInfo{ID: auth.ID, Type: "codex"}
	if errMark := poller.markAuthUnavailable(context.Background(), row, auth.Key, http.StatusUnauthorized, []byte(`{"error":"invalid"}`)); errMark != nil {
		t.Fatalf("mark unavailable: %v", errMark)
	}

	var updated models.Auth
	if errFind := conn.First(&updated, auth.ID).Error; errFind != nil {
		t.Fatalf("load auth: %v", errFind)
	}
	if updated.IsAvailable {
		t.Fatalf("expected auth to be unavailable")
	}

	var quotaCount int64
	if errCount := conn.Model(&models.Quota{}).Where("auth_id = ?", auth.ID).Count(&quotaCount).Error; errCount != nil {
		t.Fatalf("count quota: %v", errCount)
	}
	if quotaCount != 0 {
		t.Fatalf("expected quota rows deleted, got %d", quotaCount)
	}
}

func TestIsTerminalAuthStatus(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden} {
		if !isTerminalAuthStatus(status) {
			t.Fatalf("expected status %d to be terminal", status)
		}
	}
	for _, status := range []int{http.StatusTooManyRequests, http.StatusInternalServerError, http.StatusBadGateway} {
		if isTerminalAuthStatus(status) {
			t.Fatalf("expected status %d to be retryable", status)
		}
	}
}
