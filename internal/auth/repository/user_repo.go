package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	sqlcdb "sudoku-pvp/db/sqlc"
	"sudoku-pvp/internal/auth/username"
)

// ErrNotFound is returned by repository methods when the requested record does
// not exist. Callers should use errors.Is(err, repository.ErrNotFound) rather
// than comparing against storage-layer sentinel errors directly (WR-04).
var ErrNotFound = errors.New("not found")

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
// Used by GenerateUniqueUsername for collision detection in the retry loop (D-15).
func (r *UserRepo) ExistsUsername(ctx context.Context, name string) (bool, error) {
	q := sqlcdb.New(r.pool)
	exists, err := q.ExistsUsername(ctx, name)
	if err != nil {
		return false, fmt.Errorf("user repo exists username: %w", err)
	}
	return exists, nil
}

// GenerateUniqueUsername generates a unique "AdjectiveNounNumber" username with
// up to 5 collision retries. Returns an error if no unique name is found.
// Used by both CreateGuestUser and CreateGoogleUser (WR-03).
func (r *UserRepo) GenerateUniqueUsername(ctx context.Context) (string, error) {
	const maxAttempts = 5
	for i := 0; i < maxAttempts; i++ {
		candidate := username.Generate()
		exists, err := r.ExistsUsername(ctx, candidate)
		if err != nil {
			return "", fmt.Errorf("user repo generate username: %w", err)
		}
		if !exists {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("user repo: username collision after %d attempts", maxAttempts)
}

// CreateGuestUser creates a users row and wallets row in a single DB transaction.
// D-14: no user_providers row for guest accounts.
// D-15: username generated as "AdjectiveNounNumber" with up to 5 collision retries.
// T-02-03-04: BeginTx + deferred Rollback + explicit Commit ensures atomicity.
func (r *UserRepo) CreateGuestUser(ctx context.Context) (*sqlcdb.User, error) {
	uniqueName, err := r.GenerateUniqueUsername(ctx)
	if err != nil {
		return nil, fmt.Errorf("user repo create guest user: %w", err)
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("user repo guest begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	q := sqlcdb.New(tx)

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
// in a single DB transaction. It generates a unique username using the same
// retry logic as guest login (WR-03: never hardcode "Player" for Google users).
// D-12: first Google login auto-creates all three rows atomically.
// D-10: Google identity stored in user_providers, not in users table.
// T-02-03-04: BeginTx + deferred Rollback + explicit Commit ensures atomicity.
func (r *UserRepo) CreateGoogleUser(ctx context.Context, googleUserID, email string) (*sqlcdb.User, *sqlcdb.UserProvider, error) {
	uniqueName, err := r.GenerateUniqueUsername(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("user repo create google user: %w", err)
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("user repo google begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	q := sqlcdb.New(tx)

	userID := uuid.Must(uuid.NewV7())
	user, err := q.CreateUser(ctx, sqlcdb.CreateUserParams{
		ID:       toPgtypeUUID(userID),
		Username: uniqueName,
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
// WR-04: pgx.ErrNoRows is translated to ErrNotFound so callers are insulated from
// storage-layer sentinel errors.
func (r *UserRepo) GetUserProviderByProvider(ctx context.Context, provider, providerUserID string) (*sqlcdb.UserProvider, error) {
	q := sqlcdb.New(r.pool)
	result, err := q.GetUserProviderByProvider(ctx, sqlcdb.GetUserProviderByProviderParams{
		Provider:       provider,
		ProviderUserID: providerUserID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("user repo get provider: %w", err)
	}
	return &result, nil
}
