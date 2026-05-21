package usage

import (
	"context"
	"testing"
	"time"

	"github.com/router-for-me/CLIProxyAPIBusiness/internal/db"
	"github.com/router-for-me/CLIProxyAPIBusiness/internal/models"

	coreusage "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/usage"
)

func TestCalculateCostChargesCachedInputAtCacheReadPriceOnly(t *testing.T) {
	conn, errOpen := db.Open(":memory:")
	if errOpen != nil {
		t.Fatalf("open db: %v", errOpen)
	}
	if errMigrate := db.Migrate(conn); errMigrate != nil {
		t.Fatalf("migrate db: %v", errMigrate)
	}

	now := time.Now().UTC()
	var authGroup models.AuthGroup
	if errFind := conn.Where("is_default = ?", true).First(&authGroup).Error; errFind != nil {
		t.Fatalf("find default auth group: %v", errFind)
	}
	var userGroup models.UserGroup
	if errFind := conn.Where("is_default = ?", true).First(&userGroup).Error; errFind != nil {
		t.Fatalf("find default user group: %v", errFind)
	}

	inputPrice := 5.0
	outputPrice := 30.0
	cacheCreatePrice := 0.0
	cacheReadPrice := 0.5
	rule := models.BillingRule{
		AuthGroupID:           authGroup.ID,
		UserGroupID:           userGroup.ID,
		Provider:              "codex",
		Model:                 "gpt-5.5",
		BillingType:           models.BillingTypePerToken,
		PriceInputToken:       &inputPrice,
		PriceOutputToken:      &outputPrice,
		PriceCacheCreateToken: &cacheCreatePrice,
		PriceCacheReadToken:   &cacheReadPrice,
		IsEnabled:             true,
		CreatedAt:             now,
		UpdatedAt:             now,
	}
	if errCreate := conn.Create(&rule).Error; errCreate != nil {
		t.Fatalf("create billing rule: %v", errCreate)
	}

	record := coreusage.Record{
		Provider: "codex",
		Model:    "gpt-5.5",
		Detail: coreusage.Detail{
			InputTokens:  20553,
			CachedTokens: 19968,
			OutputTokens: 29,
		},
	}

	got := calculateCost(context.Background(), conn, nil, nil, nil, nil, record)
	want := int64(13779)
	if got != want {
		t.Fatalf("calculateCost() = %d, want %d", got, want)
	}
}
