package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	sqlcdb "sudoku-pvp/db/sqlc"
	"sudoku-pvp/internal/auth/username"
)

// UserRepo manages user and wallet creation in PostgreSQL.
// All mutations use DB transactions to guarantee atomic user + wallet creation.
// DEC-014: no raw SQL — all queries via sqlc-generated functions.
type UserRepo struct {
	pool *pgxpool.Pool
}

// NewUserRepo creates a new UserRepo backed by the given connection pool.
func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

// toPgtypeUUID converts a uuid.UUID value to pgtype.UUID for use with sqlc queries.
func toPgtypeUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

// ExistsUsername reports whether the given username is already taken.
// Used by CreateGuestUser for collision detection in the retry loop (D-15).
func (r *UserRepo) ExistsUsername(ctx context.Context, name string) (bool, error) {
	q := sqlcdb.New(r.pool)
	exists, err := q.ExistsUsername(ctx, name)
	if err != nil {
		return false, fmt.Errorf("user repo exists username: %w", err)
	}
	return exists, nil
}

// CreateGuestUser creates a users row and wallets row in a single DB transaction.
// D-14: no user_providers row for guest accounts.
// D-15: username generated as "AdjectiveNounNumber" with up to 5 collision retries.
// T-02-03-04: BeginTx + deferred Rollback + explicit Commit ensures atomicity.
func (r *UserRepo) CreateGuestUser(ctx context.Context) (*sqlcdb.User, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("user repo guest begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	q := sqlcdb.New(tx)

	// Collision retry loop — max 5 attempts (T-02-03-02).
	const maxAttempts = 5
	var uniqueName string
	for i := 0; i < maxAttempts; i++ {
		candidate := username.Generate()
		exists, err := q.ExistsUsername(ctx, candidate)
		if err != nil {
			return nil, fmt.Errorf("user repo guest check username: %w", err)
		}
		if !exists {
			uniqueName = candidate
			break
		}
	}
	if uniqueName == "" {
		return nil, fmt.Errorf("user repo: username collision after 5 attempts")
	}

	userID := uuid.Must(uuid.NewV7())
	user, err := q.CreateUser(ctx, sqlcdb.CreateUserParams{
		ID:       toPgtypeUUID(userID),
		Username: uniqueName,
	})
	if err != nil {
		return nil, fmt.Errorf("user repo create guest user: %w", err)
	}

	if err := q.CreateWallet(ctx, toPgtypeUUID(userID)); err != nil {
		return nil, fmt.Errorf("user repo create guest wallet: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("user repo guest commit: %w", err)
	}

	return &user, nil
}

// CreateGoogleUser creates a users row, user_providers row, and wallets row
// in a single DB transaction.
// D-12: first Google login auto-creates all three rows atomically.
// D-10: Google identity stored in user_providers, not in users table.
// T-02-03-04: BeginTx + deferred Rollback + explicit Commit ensures atomicity.
func (r *UserRepo) CreateGoogleUser(ctx context.Context, googleUserID, email, displayName string) (*sqlcdb.User, *sqlcdb.UserProvider, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("user repo google begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	q := sqlcdb.New(tx)

	userID := uuid.Must(uuid.NewV7())
	user, err := q.CreateUser(ctx, sqlcdb.CreateUserParams{
		ID:       toPgtypeUUID(userID),
		Username: displayName,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("user repo create google user: %w", err)
	}

	var emailPtr *string
	if email != "" {
		emailPtr = &email
	}

	providerID := uuid.Must(uuid.NewV7())
	provider, err := q.CreateUserProvider(ctx, sqlcdb.CreateUserProviderParams{
		ID:             toPgtypeUUID(providerID),
		UserID:         toPgtypeUUID(userID),
		Provider:       "google",
		ProviderUserID: googleUserID,
		Email:          emailPtr,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("user repo create google provider: %w", err)
	}

	if err := q.CreateWallet(ctx, toPgtypeUUID(userID)); err != nil {
		return nil, nil, fmt.Errorf("user repo create google wallet: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, fmt.Errorf("user repo google commit: %w", err)
	}

	return &user, &provider, nil
}

// GetUserProviderByProvider looks up an existing provider record for a given
// (provider, providerUserID) pair.
// D-12: called by auth service before CreateGoogleUser to detect existing Google accounts.
func (r *UserRepo) GetUserProviderByProvider(ctx context.Context, provider, providerUserID string) (*sqlcdb.UserProvider, error) {
	q := sqlcdb.New(r.pool)
	result, err := q.GetUserProviderByProvider(ctx, sqlcdb.GetUserProviderByProviderParams{
		Provider:       provider,
		ProviderUserID: providerUserID,
	})
	if err != nil {
		return nil, fmt.Errorf("user repo get provider: %w", err)
	}
	return &result, nil
}
