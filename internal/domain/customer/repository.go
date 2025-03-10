package customer

import (
	"context"

	"github.com/FACorreiaa/ink-app-backend-grpc/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
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
	return "", nil
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
