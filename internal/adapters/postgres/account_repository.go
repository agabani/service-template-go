package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/agabani/service-template-go/internal/domain"
	"github.com/agabani/service-template-go/internal/domain/account"
)

type AccountRepository struct {
	pool *pgxpool.Pool
}

func NewAccountRepository(pool *pgxpool.Pool) *AccountRepository {
	return &AccountRepository{pool: pool}
}

func (r *AccountRepository) Create(ctx context.Context, input account.CreateInput) (*account.Account, error) {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO accounts (uuid, user_id, name, currency)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, uuid, user_id, name, balance, currency, created_at, updated_at`,
		uuid.New(), input.UserID, input.Name, input.Currency,
	)
	_, a, err := scanAccount(row)
	if err != nil {
		return nil, fmt.Errorf("create account: %w", mapError(err))
	}
	return a, nil
}

func (r *AccountRepository) GetByID(ctx context.Context, id uuid.UUID) (*account.Account, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, uuid, user_id, name, balance, currency, created_at, updated_at
		 FROM accounts WHERE uuid = $1`,
		id,
	)
	_, a, err := scanAccount(row)
	if err != nil {
		return nil, fmt.Errorf("get account: %w", mapError(err))
	}
	return a, nil
}

func (r *AccountRepository) ListByUserID(ctx context.Context, userID uuid.UUID, input domain.PageInput) (domain.Page[account.Account], error) {
	limit := input.Size + 1

	var (
		rows pgx.Rows
		err  error
	)
	switch {
	case input.Before != nil:
		rows, err = r.pool.Query(ctx,
			`SELECT id, uuid, user_id, name, balance, currency, created_at, updated_at
			 FROM accounts
			 WHERE user_id = $1 AND id < $3
			 ORDER BY id DESC
			 LIMIT $2`,
			userID, limit, input.Before.ID,
		)
	case input.After != nil:
		rows, err = r.pool.Query(ctx,
			`SELECT id, uuid, user_id, name, balance, currency, created_at, updated_at
			 FROM accounts
			 WHERE user_id = $1 AND id > $3
			 ORDER BY id ASC
			 LIMIT $2`,
			userID, limit, input.After.ID,
		)
	default:
		rows, err = r.pool.Query(ctx,
			`SELECT id, uuid, user_id, name, balance, currency, created_at, updated_at
			 FROM accounts
			 WHERE user_id = $1
			 ORDER BY id ASC
			 LIMIT $2`,
			userID, limit,
		)
	}
	if err != nil {
		return domain.Page[account.Account]{}, fmt.Errorf("list accounts: %w", mapError(err))
	}
	defer rows.Close()

	accounts := make([]*account.Account, 0, limit)
	ids := make([]int64, 0, limit)
	for rows.Next() {
		internalID, a, err := scanAccount(rows)
		if err != nil {
			return domain.Page[account.Account]{}, fmt.Errorf("scan account: %w", err)
		}
		accounts = append(accounts, a)
		ids = append(ids, internalID)
	}
	if err := rows.Err(); err != nil {
		return domain.Page[account.Account]{}, fmt.Errorf("iterate accounts: %w", err)
	}

	return paginateAccounts(accounts, ids, input), nil
}

// paginateAccounts trims the over-fetched slice and derives next/prev cursors.
// For Before queries the slice arrives in DESC order and is reversed to ASC.
func paginateAccounts(accounts []*account.Account, ids []int64, input domain.PageInput) domain.Page[account.Account] {
	var next, prev *domain.PageCursor

	if input.Before != nil {
		hasMore := len(accounts) > input.Size
		if hasMore {
			accounts = accounts[:input.Size]
			ids = ids[:input.Size]
		}
		for i, j := 0, len(accounts)-1; i < j; i, j = i+1, j-1 {
			accounts[i], accounts[j] = accounts[j], accounts[i]
			ids[i], ids[j] = ids[j], ids[i]
		}
		if len(accounts) > 0 {
			next = &domain.PageCursor{ID: ids[len(ids)-1]}
			if hasMore {
				prev = &domain.PageCursor{ID: ids[0]}
			}
		}
		return domain.Page[account.Account]{Items: accounts, Next: next, Prev: prev}
	}

	if len(accounts) > input.Size {
		next = &domain.PageCursor{ID: ids[input.Size-1]}
		accounts = accounts[:input.Size]
		ids = ids[:input.Size]
	}
	if input.After != nil && len(accounts) > 0 {
		prev = &domain.PageCursor{ID: ids[0]}
	}
	return domain.Page[account.Account]{Items: accounts, Next: next, Prev: prev}
}

func (r *AccountRepository) Update(ctx context.Context, id uuid.UUID, input account.UpdateInput) (*account.Account, error) {
	row := r.pool.QueryRow(ctx,
		`UPDATE accounts
		 SET name       = COALESCE($2, name),
		     updated_at = NOW()
		 WHERE uuid = $1
		 RETURNING id, uuid, user_id, name, balance, currency, created_at, updated_at`,
		id, input.Name,
	)
	_, a, err := scanAccount(row)
	if err != nil {
		return nil, fmt.Errorf("update account: %w", mapError(err))
	}
	return a, nil
}

func (r *AccountRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.pool.Exec(ctx, `DELETE FROM accounts WHERE uuid = $1`, id)
	if err != nil {
		return fmt.Errorf("delete account: %w", mapError(err))
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("delete account: %w", mapError(pgx.ErrNoRows))
	}
	return nil
}

func scanAccount(row rowScanner) (int64, *account.Account, error) {
	var internalID int64
	var a account.Account
	if err := row.Scan(&internalID, &a.ID, &a.UserID, &a.Name, &a.Balance, &a.Currency, &a.CreatedAt, &a.UpdatedAt); err != nil {
		return 0, nil, fmt.Errorf("scan account: %w", err)
	}
	return internalID, &a, nil
}
