package adapters

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/sekai-labs/kumokura/internal/accounts/domain"
	"github.com/sekai-labs/kumokura/internal/accounts/ports"
)

type SQLiteAccountRepository struct {
	db *sql.DB
}

func NewSQLiteAccountRepository(db *sql.DB) *SQLiteAccountRepository {
	return &SQLiteAccountRepository{db: db}
}

func (r *SQLiteAccountRepository) Save(ctx context.Context, acc *domain.Account) error {
	query := `
INSERT INTO accounts (id, name, account_type, endpoint, region, use_path_style, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
`
	usePathStyle := 0
	if acc.UsePathStyle {
		usePathStyle = 1
	}

	_, err := r.db.ExecContext(ctx, query,
		string(acc.ID),
		acc.Name,
		string(acc.Type),
		acc.Endpoint,
		acc.Region,
		usePathStyle,
		acc.CreatedAt.Format(time.RFC3339Nano),
		acc.UpdatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("insert account: %w", err)
	}
	return nil
}

func (r *SQLiteAccountRepository) FindByID(ctx context.Context, id domain.AccountID) (*domain.Account, error) {
	query := `
SELECT id, name, account_type, endpoint, region, use_path_style, created_at, updated_at
FROM accounts
WHERE id = ?
`
	var (
		accID        string
		name         string
		accType      string
		endpoint     string
		region       string
		usePathStyle int
		createdAtStr string
		updatedAtStr string
	)

	err := r.db.QueryRowContext(ctx, query, string(id)).Scan(
		&accID,
		&name,
		&accType,
		&endpoint,
		&region,
		&usePathStyle,
		&createdAtStr,
		&updatedAtStr,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ports.ErrAccountNotFound
		}
		return nil, fmt.Errorf("find account by id: %w", err)
	}

	createdAt, _ := time.Parse(time.RFC3339Nano, createdAtStr)
	if createdAt.IsZero() {
		createdAt, _ = time.Parse("2006-01-02 15:04:05", createdAtStr)
	}
	updatedAt, _ := time.Parse(time.RFC3339Nano, updatedAtStr)
	if updatedAt.IsZero() {
		updatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAtStr)
	}

	return &domain.Account{
		ID:           domain.AccountID(accID),
		Name:         name,
		Type:         domain.AccountType(accType),
		Endpoint:     endpoint,
		Region:       region,
		UsePathStyle: usePathStyle == 1,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}, nil
}

func (r *SQLiteAccountRepository) FindByName(ctx context.Context, name string) (*domain.Account, error) {
	query := `
SELECT id, name, account_type, endpoint, region, use_path_style, created_at, updated_at
FROM accounts
WHERE name = ?
`
	var (
		accID        string
		accName      string
		accType      string
		endpoint     string
		region       string
		usePathStyle int
		createdAtStr string
		updatedAtStr string
	)

	err := r.db.QueryRowContext(ctx, query, name).Scan(
		&accID,
		&accName,
		&accType,
		&endpoint,
		&region,
		&usePathStyle,
		&createdAtStr,
		&updatedAtStr,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ports.ErrAccountNotFound
		}
		return nil, fmt.Errorf("find account by name: %w", err)
	}

	createdAt, _ := time.Parse(time.RFC3339Nano, createdAtStr)
	if createdAt.IsZero() {
		createdAt, _ = time.Parse("2006-01-02 15:04:05", createdAtStr)
	}
	updatedAt, _ := time.Parse(time.RFC3339Nano, updatedAtStr)
	if updatedAt.IsZero() {
		updatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAtStr)
	}

	return &domain.Account{
		ID:           domain.AccountID(accID),
		Name:         accName,
		Type:         domain.AccountType(accType),
		Endpoint:     endpoint,
		Region:       region,
		UsePathStyle: usePathStyle == 1,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}, nil
}

func (r *SQLiteAccountRepository) FindAll(ctx context.Context) ([]*domain.Account, error) {
	query := `
SELECT id, name, account_type, endpoint, region, use_path_style, created_at, updated_at
FROM accounts
ORDER BY name ASC
`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list accounts: %w", err)
	}
	defer rows.Close()

	var accounts []*domain.Account
	for rows.Next() {
		var (
			accID        string
			name         string
			accType      string
			endpoint     string
			region       string
			usePathStyle int
			createdAtStr string
			updatedAtStr string
		)

		if err := rows.Scan(
			&accID,
			&name,
			&accType,
			&endpoint,
			&region,
			&usePathStyle,
			&createdAtStr,
			&updatedAtStr,
		); err != nil {
			return nil, fmt.Errorf("scan account: %w", err)
		}

		createdAt, _ := time.Parse(time.RFC3339Nano, createdAtStr)
		if createdAt.IsZero() {
			createdAt, _ = time.Parse("2006-01-02 15:04:05", createdAtStr)
		}
		updatedAt, _ := time.Parse(time.RFC3339Nano, updatedAtStr)
		if updatedAt.IsZero() {
			updatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAtStr)
		}

		accounts = append(accounts, &domain.Account{
			ID:           domain.AccountID(accID),
			Name:         name,
			Type:         domain.AccountType(accType),
			Endpoint:     endpoint,
			Region:       region,
			UsePathStyle: usePathStyle == 1,
			CreatedAt:    createdAt,
			UpdatedAt:    updatedAt,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate accounts: %w", err)
	}

	return accounts, nil
}

func (r *SQLiteAccountRepository) Update(ctx context.Context, acc *domain.Account) error {
	query := `
UPDATE accounts
SET name = ?, endpoint = ?, region = ?, use_path_style = ?, updated_at = ?
WHERE id = ?
`
	usePathStyle := 0
	if acc.UsePathStyle {
		usePathStyle = 1
	}

	res, err := r.db.ExecContext(ctx, query,
		acc.Name,
		acc.Endpoint,
		acc.Region,
		usePathStyle,
		acc.UpdatedAt.Format(time.RFC3339Nano),
		string(acc.ID),
	)
	if err != nil {
		return fmt.Errorf("update account: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("check affected rows: %w", err)
	}
	if rows == 0 {
		return ports.ErrAccountNotFound
	}

	return nil
}

func (r *SQLiteAccountRepository) Delete(ctx context.Context, id domain.AccountID) error {
	query := `DELETE FROM accounts WHERE id = ?`
	res, err := r.db.ExecContext(ctx, query, string(id))
	if err != nil {
		return fmt.Errorf("delete account: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("check affected rows: %w", err)
	}
	if rows == 0 {
		return ports.ErrAccountNotFound
	}

	return nil
}
