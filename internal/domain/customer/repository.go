package customer

import (
	"context"
	"fmt"
	"strings"

	"github.com/FACorreiaa/ink-app-backend-grpc/internal/domain"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type PostgresCustomerRepository struct {
	pgpool *pgxpool.Pool
	redis  *redis.Client
}

// NewRepository creates a new AuthService
func NewRepository(db *pgxpool.Pool, redis *redis.Client) *PostgresCustomerRepository {
	return &PostgresCustomerRepository{pgpool: db, redis: redis}
}

func (r *PostgresCustomerRepository) Create(ctx context.Context, customer *domain.Customer) (string, error) {
	tx, err := r.pgpool.Begin(ctx)
	if err != nil {
		return "", status.Error(codes.Internal, "could not start transaction")
	}
	defer tx.Rollback(ctx)

	if customer == nil {
		return "", status.Error(codes.InvalidArgument, "customer is required")
	}

	// Format full name if not provided
	fullName := customer.FullName
	if fullName == "" && (customer.FirstName != "" || customer.LastName != "") {
		fullName = strings.TrimSpace(fmt.Sprintf("%s %s", customer.FirstName, customer.LastName))
	}

	// Format birthday as string if available
	var birthdayStr *string
	if !customer.DateOfBirth.IsZero() {
		bdayStr := customer.DateOfBirth.Format("2006-01-02")
		birthdayStr = &bdayStr
	}

	var customerID string
	query := `INSERT INTO customers (
		studio_id, full_name, email, phone, notes, nif, address,
		city, postal_code, country, id_card_number, first_name, last_name, birthday, 
		is_archived, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, NOW(), NOW())
		RETURNING id`

	err = tx.QueryRow(ctx, query,
		customer.StudioID,
		fullName,
		customer.Email,
		customer.Phone,
		customer.Notes,
		customer.NIF,
		customer.Address,
		customer.City,
		customer.PostalCode,
		customer.Country,
		customer.IDCardNumber,
		customer.FirstName,
		customer.LastName,
		birthdayStr,
		customer.IsArchived,
	).Scan(&customerID)

	if err != nil {
		// Check for unique constraint violations
		if pgErr, ok := err.(*pgconn.PgError); ok {
			if pgErr.Code == "23505" { // unique_violation
				if strings.Contains(pgErr.Message, "email") {
					return "", status.Error(codes.AlreadyExists, "a customer with this email already exists")
				}
				if strings.Contains(pgErr.Message, "phone") {
					return "", status.Error(codes.AlreadyExists, "a customer with this phone already exists")
				}
			}
		}
		return "", status.Errorf(codes.Internal, "failed to create customer: %v", err)
	}

	// If we got here, commit the transaction
	if err = tx.Commit(ctx); err != nil {
		return "", status.Errorf(codes.Internal, "failed to commit transaction: %v", err)
	}

	return customerID, nil
}

func (r *PostgresCustomerRepository) Get(ctx context.Context, id string) (*domain.Customer, error) {
	return nil, nil
}

func (r *PostgresCustomerRepository) GetByID(ctx context.Context, id string) (*domain.Customer, error) {
	return nil, nil
}

func (r *PostgresCustomerRepository) Update(ctx context.Context, customer *domain.Customer) error {
	return nil
}

func (r *PostgresCustomerRepository) Delete(ctx context.Context, id string) error {
	return nil
}

func (r *PostgresCustomerRepository) Archive(ctx context.Context, id string) error {
	return nil
}

func (r *PostgresCustomerRepository) List(ctx context.Context, filter domain.CustomerFilter) (domain.PagedResult[domain.Customer], error) {
	return domain.PagedResult[domain.Customer]{
		Items:      []domain.Customer{},
		TotalCount: 0,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
	}, nil
}

func (r *PostgresCustomerRepository) GetByEmail(ctx context.Context, email string) (*domain.Customer, error) {
	return nil, nil
}

func (r *PostgresCustomerRepository) GetByPhone(ctx context.Context, phone string) (*domain.Customer, error) {
	return nil, nil
}

func (r *PostgresCustomerRepository) AddHistory(ctx context.Context, history *domain.CustomerHistory) error {
	return nil
}

func (r *PostgresCustomerRepository) GetHistory(ctx context.Context, customerID string) ([]*domain.CustomerHistory, error) {
	return nil, nil
}

func (r *PostgresCustomerRepository) AddNote(ctx context.Context, note *domain.CustomerNote) error {
	return nil
}

func (r *PostgresCustomerRepository) GetNotes(ctx context.Context, customerID string) ([]*domain.CustomerNote, error) {
	return nil, nil
}

func (r *PostgresCustomerRepository) Search(ctx context.Context, filter domain.CustomerFilter) (domain.PagedResult[domain.Customer], error) {
	return domain.PagedResult[domain.Customer]{
		Items:      []domain.Customer{},
		TotalCount: 0,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
	}, nil
}
func (r *PostgresCustomerRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	return false, nil
}

func (r *PostgresCustomerRepository) ExistsByPhone(ctx context.Context, phone string) (bool, error) {
	return false, nil
}
