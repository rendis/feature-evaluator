package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/rendis/feature-evaluator/internal/domain/member"
	"github.com/rendis/feature-evaluator/pkg/apierror"
)

// MemberRepo implements member.Repository using PostgreSQL.
type MemberRepo struct {
	client *Client
}

// NewMemberRepo creates a new MemberRepo.
func NewMemberRepo(client *Client) *MemberRepo {
	return &MemberRepo{client: client}
}

// Create inserts a new member.
func (r *MemberRepo) Create(ctx context.Context, m *member.Member) error {
	if m.ID == "" {
		id, err := newID()
		if err != nil {
			return err
		}
		m.ID = id
	}
	m.WorkspaceKey = wsKey(ctx)

	_, err := r.client.db(ctx).Exec(ctx, `
		INSERT INTO members (
			id, workspace_key, email, role, display_name, added_by, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, m.ID, m.WorkspaceKey, m.Email, m.Role, m.DisplayName, m.AddedBy, m.CreatedAt, m.UpdatedAt)
	if isUniqueViolation(err) {
		return apierror.NewConflict(
			fmt.Sprintf("member with email %q already exists", m.Email),
			"error.memberEmailExists",
		)
	}
	if err != nil {
		return fmt.Errorf("insert member: %w", err)
	}

	return nil
}

// GetByID finds a member by ID.
func (r *MemberRepo) GetByID(ctx context.Context, id string) (*member.Member, error) {
	parsed, err := parseUUID(id)
	if err != nil {
		return nil, apierror.NewBadRequest("invalid member ID", "error.invalidId")
	}

	row := r.client.db(ctx).QueryRow(ctx, `
		SELECT id, workspace_key, email, role, display_name, added_by, created_at, updated_at
		FROM members
		WHERE workspace_key = $1 AND id = $2
	`, wsKey(ctx), parsed)

	var m member.Member
	if err := row.Scan(
		&m.ID,
		&m.WorkspaceKey,
		&m.Email,
		&m.Role,
		&m.DisplayName,
		&m.AddedBy,
		&m.CreatedAt,
		&m.UpdatedAt,
	); err != nil {
		if isNoRows(err) {
			return nil, apierror.NewNotFound("member not found", "error.memberNotFound")
		}
		return nil, fmt.Errorf("find member by id: %w", err)
	}

	return &m, nil
}

// GetByEmail finds a member by email.
func (r *MemberRepo) GetByEmail(ctx context.Context, email string) (*member.Member, error) {
	row := r.client.db(ctx).QueryRow(ctx, `
		SELECT id, workspace_key, email, role, display_name, added_by, created_at, updated_at
		FROM members
		WHERE workspace_key = $1 AND email = $2
	`, wsKey(ctx), strings.ToLower(strings.TrimSpace(email)))

	var m member.Member
	if err := row.Scan(
		&m.ID,
		&m.WorkspaceKey,
		&m.Email,
		&m.Role,
		&m.DisplayName,
		&m.AddedBy,
		&m.CreatedAt,
		&m.UpdatedAt,
	); err != nil {
		if isNoRows(err) {
			return nil, apierror.NewNotFound("member not found", "error.memberNotFound")
		}
		return nil, fmt.Errorf("find member by email: %w", err)
	}

	return &m, nil
}

// Update updates a member.
func (r *MemberRepo) Update(ctx context.Context, m *member.Member) error {
	parsed, err := parseUUID(m.ID)
	if err != nil {
		return apierror.NewBadRequest("invalid member ID", "error.invalidId")
	}

	tag, err := r.client.db(ctx).Exec(ctx, `
		UPDATE members
		SET display_name = $3, role = $4, updated_at = $5
		WHERE workspace_key = $1 AND id = $2
	`, wsKey(ctx), parsed, m.DisplayName, m.Role, m.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update member: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apierror.NewNotFound("member not found", "error.memberNotFound")
	}

	return nil
}

// UpdateRole updates a member role.
func (r *MemberRepo) UpdateRole(ctx context.Context, id string, role member.Role) error {
	parsed, err := parseUUID(id)
	if err != nil {
		return apierror.NewBadRequest("invalid member ID", "error.invalidId")
	}

	tag, err := r.client.db(ctx).Exec(ctx, `
		UPDATE members
		SET role = $3, updated_at = NOW()
		WHERE workspace_key = $1 AND id = $2
	`, wsKey(ctx), parsed, role)
	if err != nil {
		return fmt.Errorf("update member role: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apierror.NewNotFound("member not found", "error.memberNotFound")
	}

	return nil
}

// Delete removes a member by ID.
func (r *MemberRepo) Delete(ctx context.Context, id string) error {
	parsed, err := parseUUID(id)
	if err != nil {
		return apierror.NewBadRequest("invalid member ID", "error.invalidId")
	}

	tag, err := r.client.db(ctx).Exec(ctx, `
		DELETE FROM members
		WHERE workspace_key = $1 AND id = $2
	`, wsKey(ctx), parsed)
	if err != nil {
		return fmt.Errorf("delete member: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apierror.NewNotFound("member not found", "error.memberNotFound")
	}

	return nil
}

// List returns all members in the workspace.
func (r *MemberRepo) List(ctx context.Context) ([]member.Member, error) {
	rows, err := r.client.db(ctx).Query(ctx, `
		SELECT id, workspace_key, email, role, display_name, added_by, created_at, updated_at
		FROM members
		WHERE workspace_key = $1
		ORDER BY created_at ASC
	`, wsKey(ctx))
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	defer rows.Close()

	members := make([]member.Member, 0)
	for rows.Next() {
		var m member.Member
		if err := rows.Scan(
			&m.ID,
			&m.WorkspaceKey,
			&m.Email,
			&m.Role,
			&m.DisplayName,
			&m.AddedBy,
			&m.CreatedAt,
			&m.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan member: %w", err)
		}
		members = append(members, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate members: %w", err)
	}

	if members == nil {
		return []member.Member{}, nil
	}
	return members, nil
}

// CountAll counts all members in the workspace.
func (r *MemberRepo) CountAll(ctx context.Context) (int64, error) {
	var count int64
	if err := r.client.db(ctx).QueryRow(ctx, `
		SELECT COUNT(*)
		FROM members
		WHERE workspace_key = $1
	`, wsKey(ctx)).Scan(&count); err != nil {
		return 0, fmt.Errorf("count members: %w", err)
	}

	return count, nil
}

// CountByRole counts members by role in the workspace.
func (r *MemberRepo) CountByRole(ctx context.Context, role member.Role) (int64, error) {
	var count int64
	if err := r.client.db(ctx).QueryRow(ctx, `
		SELECT COUNT(*)
		FROM members
		WHERE workspace_key = $1 AND role = $2
	`, wsKey(ctx), role).Scan(&count); err != nil {
		return 0, fmt.Errorf("count members by role: %w", err)
	}

	return count, nil
}

// TransferOwnership atomically switches owner/admin roles between members.
func (r *MemberRepo) TransferOwnership(ctx context.Context, fromID, toID string) error {
	fromUUID, err := parseUUID(fromID)
	if err != nil {
		return apierror.NewBadRequest("invalid source member ID", "error.invalidId")
	}
	toUUID, err := parseUUID(toID)
	if err != nil {
		return apierror.NewBadRequest("invalid target member ID", "error.invalidId")
	}

	return r.client.WithinTx(ctx, func(txCtx context.Context) error {
		if _, err := r.client.db(txCtx).Exec(txCtx, `
			UPDATE members
			SET role = $3, updated_at = NOW()
			WHERE workspace_key = $1 AND id = $2
		`, wsKey(txCtx), fromUUID, member.RoleAdmin); err != nil {
			return fmt.Errorf("demote owner: %w", err)
		}

		tag, err := r.client.db(txCtx).Exec(txCtx, `
			UPDATE members
			SET role = $3, updated_at = NOW()
			WHERE workspace_key = $1 AND id = $2
		`, wsKey(txCtx), toUUID, member.RoleOwner)
		if err != nil {
			return fmt.Errorf("promote owner: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return apierror.NewNotFound("member not found", "error.memberNotFound")
		}

		return nil
	})
}
