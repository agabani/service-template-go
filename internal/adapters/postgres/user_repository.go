package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/agabani/service-template-go/internal/domain"
	"github.com/agabani/service-template-go/internal/domain/user"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) Create(ctx context.Context, input user.CreateInput) (*user.User, error) {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO users (uuid, email, name) VALUES ($1, $2, $3)
		 RETURNING id, uuid, email, name, created_at, updated_at`,
		uuid.New(), input.Email, input.Name,
	)
	_, u, err := scanUser(row)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", mapError(err))
	}
	return u, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, uuid, email, name, created_at, updated_at FROM users WHERE uuid = $1`,
		id,
	)
	_, u, err := scanUser(row)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", mapError(err))
	}
	return u, nil
}

func (r *UserRepository) List(ctx context.Context, input domain.PageInput) (domain.Page[user.User], error) {
	limit := input.Size + 1

	var (
		rows pgx.Rows
		err  error
	)
	switch {
	case input.Before != nil:
		rows, err = r.pool.Query(ctx,
			`SELECT id, uuid, email, name, created_at, updated_at
			 FROM users
			 WHERE id < $2
			 ORDER BY id DESC
			 LIMIT $1`,
			limit, input.Before.ID,
		)
	case input.After != nil:
		rows, err = r.pool.Query(ctx,
			`SELECT id, uuid, email, name, created_at, updated_at
			 FROM users
			 WHERE id > $2
			 ORDER BY id ASC
			 LIMIT $1`,
			limit, input.After.ID,
		)
	default:
		rows, err = r.pool.Query(ctx,
			`SELECT id, uuid, email, name, created_at, updated_at
			 FROM users
			 ORDER BY id ASC
			 LIMIT $1`,
			limit,
		)
	}
	if err != nil {
		return domain.Page[user.User]{}, fmt.Errorf("list users: %w", mapError(err))
	}
	defer rows.Close()

	users := make([]*user.User, 0, limit)
	ids := make([]int64, 0, limit)
	for rows.Next() {
		internalID, u, err := scanUser(rows)
		if err != nil {
			return domain.Page[user.User]{}, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, u)
		ids = append(ids, internalID)
	}
	if err := rows.Err(); err != nil {
		return domain.Page[user.User]{}, fmt.Errorf("iterate users: %w", err)
	}

	return paginateUsers(users, ids, input), nil
}

// paginateUsers trims the over-fetched slice and derives next/prev cursors.
// For Before queries the slice arrives in DESC order and is reversed to ASC.
func paginateUsers(users []*user.User, ids []int64, input domain.PageInput) domain.Page[user.User] {
	var next, prev *domain.PageCursor

	if input.Before != nil {
		hasMore := len(users) > input.Size
		if hasMore {
			users = users[:input.Size]
			ids = ids[:input.Size]
		}
		for i, j := 0, len(users)-1; i < j; i, j = i+1, j-1 {
			users[i], users[j] = users[j], users[i]
			ids[i], ids[j] = ids[j], ids[i]
		}
		if len(users) > 0 {
			next = &domain.PageCursor{ID: ids[len(ids)-1]}
			if hasMore {
				prev = &domain.PageCursor{ID: ids[0]}
			}
		}
		return domain.Page[user.User]{Items: users, Next: next, Prev: prev}
	}

	if len(users) > input.Size {
		next = &domain.PageCursor{ID: ids[input.Size-1]}
		users = users[:input.Size]
		ids = ids[:input.Size]
	}
	if input.After != nil && len(users) > 0 {
		prev = &domain.PageCursor{ID: ids[0]}
	}
	return domain.Page[user.User]{Items: users, Next: next, Prev: prev}
}

func (r *UserRepository) Update(ctx context.Context, id uuid.UUID, input user.UpdateInput) (*user.User, error) {
	row := r.pool.QueryRow(ctx,
		`UPDATE users
		 SET email      = COALESCE($2, email),
		     name       = COALESCE($3, name),
		     updated_at = NOW()
		 WHERE uuid = $1
		 RETURNING id, uuid, email, name, created_at, updated_at`,
		id, input.Email, input.Name,
	)
	_, u, err := scanUser(row)
	if err != nil {
		return nil, fmt.Errorf("update user: %w", mapError(err))
	}
	return u, nil
}

func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.pool.Exec(ctx, `DELETE FROM users WHERE uuid = $1`, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", mapError(err))
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("delete user: %w", mapError(pgx.ErrNoRows))
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanUser(row rowScanner) (int64, *user.User, error) {
	var internalID int64
	var u user.User
	if err := row.Scan(&internalID, &u.ID, &u.Email, &u.Name, &u.CreatedAt, &u.UpdatedAt); err != nil {
		return 0, nil, fmt.Errorf("scan user: %w", err)
	}
	return internalID, &u, nil
}
