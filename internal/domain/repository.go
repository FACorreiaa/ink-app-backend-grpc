package domain

import (
	"context"
	"time"
)

// type AuthRepository interface {
// 	Register(ctx context.Context, req *upb.RegisterRequest) (*upb.RegisterResponse, error)
// 	Login(ctx context.Context, req *upb.LoginRequest) (*upb.LoginResponse, error)
// 	Logout(ctx context.Context, req *upb.NilReq) (*upb.NilRes, error)
// 	ChangePassword(ctx context.Context, req *upb.ChangePasswordRequest) (*upb.ChangePasswordResponse, error)
// 	ChangeEmail(ctx context.Context, req *upb.ChangeEmailRequest) (*upb.ChangeEmailResponse, error)

// 	// GetAllUsers Users Methods
// 	GetAllUsers(ctx context.Context) (*upb.GetAllUsersResponse, error)
// 	GetUserByID(ctx context.Context, req *upb.GetUserByIDRequest) (*upb.GetUserByIDResponse, error)
// 	DeleteUser(ctx context.Context, req *upb.DeleteUserRequest) (*upb.DeleteUserResponse, error)
// 	UpdateUser(ctx context.Context, req *upb.UpdateUserRequest) (*upb.UpdateUserResponse, error)
// 	InsertUser(ctx context.Context, req *upb.InsertUserRequest) (*upb.InsertUserResponse, error)
// }

// DB Layer methods later?

// type CustomerService interface {
// 	CreateCustomer(ctx context.Context, req *upc.CreateCustomerRequest) (*upc.CreateCustomerResponse, error)
// 	GetCustomer(ctx context.Context, req *upc.GetCustomerRequest) (*upc.GetCustomerResponse, error)
// 	ListCustomers(ctx context.Context, req *upc.ListCustomersRequest) (*upc.ListCustomersResponse, error)
// 	UpdateCustomer(ctx context.Context, req *upc.UpdateCustomerRequest) (*upc.UpdateCustomerResponse, error)

// 	DeleteCustomer(ctx context.Context, req *upc.DeleteCustomerRequest) (*upc.DeleteCustomerResponse, error)
// 	ArchiveCustomer(ctx context.Context, req *upc.ArchiveCustomerRequest) (*upc.ArchiveCustomerResponse, error)
// 	GetCustomerHistory(ctx context.Context, req *upc.GetCustomerHistoryRequest) (*upc.GetCustomerHistoryResponse, error)
// 	AddCustomerNote(ctx context.Context, req *upc.AddCustomerNoteRequest) (*upc.AddCustomerNoteResponse, error)
// 	SearchCustomers(ctx context.Context, req *upc.SearchCustomersRequest) (*upc.SearchCustomersResponse, error)
// }

// Customer represents the customer entity in your domain
type Customer struct {
	ID           string
	StudioID     string
	FullName     string
	FirstName    string
	LastName     string
	Email        string
	Phone        string
	Notes        string
	NIF          string
	Address      string
	City         string
	PostalCode   string
	Country      string
	IDCardNumber string
	DateOfBirth  time.Time
	IsArchived   bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
}

// CustomerNote represents a note attached to a customer
type CustomerNote struct {
	ID         string
	CustomerID string
	Content    string
	CreatedBy  string
	CreatedAt  time.Time
}

// CustomerHistory represents a historical interaction with a customer
type CustomerHistory struct {
	ID          string
	CustomerID  string
	Type        string // e.g., "appointment", "message", "purchase"
	Description string
	Timestamp   time.Time
}

// CustomerFilter defines search criteria for customers
type CustomerFilter struct {
	Name            string
	Email           string
	Phone           string
	DateOfBirth     *time.Time
	CreatedFrom     *time.Time
	CreatedTo       *time.Time
	IncludeArchived bool
	Page            int
	PageSize        int
}

// PagedResult is a generic paged result for list operations
type PagedResult[T any] struct {
	Items      []T
	TotalCount int64
	Page       int
	PageSize   int
}

type CustomerRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, customer *Customer) (string, error)
	GetByID(ctx context.Context, id string) (*Customer, error)
	List(ctx context.Context, filter CustomerFilter) (PagedResult[Customer], error)
	Update(ctx context.Context, customer *Customer) error

	// Extended operations
	Delete(ctx context.Context, id string) error
	Archive(ctx context.Context, id string) error

	// History operations
	AddHistory(ctx context.Context, history *CustomerHistory) error
	GetHistory(ctx context.Context, customerID string) ([]*CustomerHistory, error)

	// Note operations
	AddNote(ctx context.Context, note *CustomerNote) error
	GetNotes(ctx context.Context, customerID string) ([]*CustomerNote, error)

	// Search operation
	Search(ctx context.Context, filter CustomerFilter) (PagedResult[Customer], error)

	// Additional helper functions
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	ExistsByPhone(ctx context.Context, phone string) (bool, error)
}
