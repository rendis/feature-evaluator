package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/rendis/feature-evaluator/internal/domain/changelog"
)

// ChangelogRepo implements changelog.Repository using PostgreSQL.
type ChangelogRepo struct {
	client *Client
}

// NewChangelogRepo creates a new ChangelogRepo.
func NewChangelogRepo(client *Client) *ChangelogRepo {
	return &ChangelogRepo{client: client}
}

// Create inserts a changelog entry.
func (r *ChangelogRepo) Create(ctx context.Context, entry *changelog.ChangeEntry) error {
	if entry.ID == "" {
		id, err := newID()
		if err != nil {
			return err
		}
		entry.ID = id
	}
	entry.WorkspaceKey = wsKey(ctx)

	fieldChangesJSON, err := jsonBytes(entry.FieldChanges, "[]")
	if err != nil {
		return fmt.Errorf("marshal changelog field changes: %w", err)
	}
	metadataJSON, err := jsonBytes(entry.Metadata, "{}")
	if err != nil {
		return fmt.Errorf("marshal changelog metadata: %w", err)
	}

	_, err = r.client.db(ctx).Exec(ctx, `
		INSERT INTO changelog (
			id, workspace_key, entity_type, entity_key, parent_key, action,
			actor, actor_type, field_changes, metadata, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9::jsonb, $10::jsonb, $11
		)
	`,
		entry.ID,
		entry.WorkspaceKey,
		entry.EntityType,
		entry.EntityKey,
		entry.ParentKey,
		entry.Action,
		entry.Actor,
		entry.ActorType,
		fieldChangesJSON,
		metadataJSON,
		entry.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert changelog: %w", err)
	}

	return nil
}

// List returns paginated changelog entries.
func (r *ChangelogRepo) List(ctx context.Context, params changelog.ListParams) (*changelog.ListResult, error) {
	return r.query(ctx, params, "", "")
}

// ListByEntity returns paginated changelog entries for an entity.
func (r *ChangelogRepo) ListByEntity(ctx context.Context, entityType, entityKey string, params changelog.ListParams) (*changelog.ListResult, error) {
	return r.query(ctx, params, entityType, entityKey)
}

func (r *ChangelogRepo) query(ctx context.Context, params changelog.ListParams, forcedEntityType, forcedEntityKey string) (*changelog.ListResult, error) {
	where := []string{"workspace_key = $1"}
	args := []any{wsKey(ctx)}
	arg := 2

	if forcedEntityType != "" {
		where = append(where, fmt.Sprintf("entity_type = $%d", arg))
		args = append(args, forcedEntityType)
		arg++
	} else if params.EntityType != "" {
		where = append(where, fmt.Sprintf("entity_type = $%d", arg))
		args = append(args, params.EntityType)
		arg++
	}
	if forcedEntityKey != "" {
		where = append(where, fmt.Sprintf("entity_key = $%d", arg))
		args = append(args, forcedEntityKey)
		arg++
	} else if params.EntityKey != "" {
		where = append(where, fmt.Sprintf("entity_key = $%d", arg))
		args = append(args, params.EntityKey)
		arg++
	}
	if params.Actor != "" {
		where = append(where, fmt.Sprintf("actor = $%d", arg))
		args = append(args, params.Actor)
		arg++
	}
	if params.Action != "" {
		where = append(where, fmt.Sprintf("action = $%d", arg))
		args = append(args, params.Action)
		arg++
	}

	fromPtr, toPtr := parseCreatedAtFilters(params.From, params.To)
	if fromPtr != nil {
		where = append(where, fmt.Sprintf("created_at >= $%d", arg))
		args = append(args, *fromPtr)
		arg++
	}
	if toPtr != nil {
		where = append(where, fmt.Sprintf("created_at <= $%d", arg))
		args = append(args, *toPtr)
		arg++
	}

	predicate := strings.Join(where, " AND ")

	var total int64
	if err := r.client.db(ctx).QueryRow(ctx, `SELECT COUNT(*) FROM changelog WHERE `+predicate, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count changelog entries: %w", err)
	}

	args = append(args, params.PageSize, (params.Page-1)*params.PageSize)
	rows, err := r.client.db(ctx).Query(ctx, `
		SELECT id, workspace_key, entity_type, entity_key, parent_key, action,
		       actor, actor_type, field_changes, metadata, created_at
		FROM changelog
		WHERE `+predicate+`
		ORDER BY created_at DESC
		LIMIT $`+fmt.Sprint(arg)+` OFFSET $`+fmt.Sprint(arg+1), args...)
	if err != nil {
		return nil, fmt.Errorf("list changelog entries: %w", err)
	}
	defer rows.Close()

	entries := make([]changelog.ChangeEntry, 0)
	for rows.Next() {
		entry, err := scanChangelog(rows)
		if err != nil {
			return nil, fmt.Errorf("scan changelog entry: %w", err)
		}
		entries = append(entries, *entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate changelog entries: %w", err)
	}

	return &changelog.ListResult{
		Data:       entries,
		Total:      total,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: calcTotalPages(total, params.PageSize),
	}, nil
}

type changelogScanner interface {
	Scan(dest ...any) error
}

func scanChangelog(scanner changelogScanner) (*changelog.ChangeEntry, error) {
	var entry changelog.ChangeEntry
	var fieldChangesJSON []byte
	var metadataJSON []byte
	if err := scanner.Scan(
		&entry.ID,
		&entry.WorkspaceKey,
		&entry.EntityType,
		&entry.EntityKey,
		&entry.ParentKey,
		&entry.Action,
		&entry.Actor,
		&entry.ActorType,
		&fieldChangesJSON,
		&metadataJSON,
		&entry.CreatedAt,
	); err != nil {
		return nil, err
	}
	if err := decodeJSON(fieldChangesJSON, &entry.FieldChanges); err != nil {
		return nil, err
	}
	if err := decodeJSON(metadataJSON, &entry.Metadata); err != nil {
		return nil, err
	}

	return &entry, nil
}
