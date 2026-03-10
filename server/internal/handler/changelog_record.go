package handler

import (
	"context"
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/rendis/feature-evaluator/internal/domain/changelog"
	"github.com/rendis/feature-evaluator/internal/domain/workspace"
	"github.com/rendis/feature-evaluator/internal/server/middleware"
)

// recordChange fires a changelog entry in a background goroutine.
// Safe to call with nil changelogSvc (no-op).
func recordChange(c *gin.Context, svc *changelog.Service, entry *changelog.ChangeEntry) {
	if svc == nil {
		return
	}

	entry.Actor = middleware.GetUserEmail(c)
	if middleware.GetAuthMethod(c) == middleware.AuthMethodAPIKey {
		entry.ActorType = changelog.ActorAPIKey
	} else {
		entry.ActorType = changelog.ActorUser
	}

	// Carry workspace key into the background goroutine so the repo
	// correctly scopes the inserted changelog entry.
	bgCtx := workspace.WithKey(context.Background(), middleware.GetWorkspaceKey(c))

	go func() {
		if err := svc.Record(bgCtx, entry); err != nil {
			slog.Error("recording changelog", "error", err,
				"entityType", entry.EntityType,
				"entityKey", entry.EntityKey,
				"action", entry.Action,
			)
		}
	}()
}
