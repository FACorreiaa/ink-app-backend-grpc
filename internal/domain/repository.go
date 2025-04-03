package domain

import (
	"context"
	"time"

	"google.golang.org/protobuf/types/known/fieldmaskpb"
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

type StudioSession struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Tenant   string `json:"tenant"`
}

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

// StaffMember represents a staff member in a studio
type OwnerInfo struct {
	ID          string
	Email       string
	Password    string
	DisplayName string
	Username    string
	FirstName   string
	LastName    string
}

type Studio struct {
	ID        string
	Name      string
	Address   string
	Phone     string
	Email     string
	Website   string
	Tenant    string
	Owner     *OwnerInfo
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

type StaffMember struct {
	ID          string
	StudioID    string
	UserID      string
	Role        string
	Permissions []string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}

type User struct {
	ID           string
	Username     string
	Email        string
	PasswordHash string
	Role         string // "owner", "staff", "customer", etc.
	StudioID     string // Which studio they belong to (if applicable)
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
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

type StudioAuthRepository interface {
	Login(ctx context.Context, tenant, email, password string) (string, string, error)
	Logout(ctx context.Context, tenant, sessionID string) error
	GetSession(ctx context.Context, tenant, sessionID string) (*StudioSession, error)
	RefreshSession(ctx context.Context, tenant, refreshToken string) (string, string, error) // accessToken, refreshToken, error
	Register(ctx context.Context, tenant, username, email, password, role string) error
	ValidateCredentials(ctx context.Context, tenant, email, password string) (bool, error)
	ValidateSession(ctx context.Context, tenant, sessionID string) (bool, error)
}

type StudioRepository interface {
	CreateStudio(ctx context.Context, tenant string, studio *Studio, owner *OwnerInfo) (string, error) // studioID, error
	UpdateStudio(ctx context.Context, tenant, studioID string, studio *Studio, updateMask *fieldmaskpb.FieldMask) error
	GetStudio(ctx context.Context, tenant, studioID string) (*Studio, error)
	ListStudios(ctx context.Context, tenant string, pageSize, pageNumber int32, filter, ownerID string) ([]*Studio, int32, error) // studios, totalCount, error

	// Staff management
	AddStaffMember(ctx context.Context, tenant, studioID, userID, role string, permissions []string) (string, error) // staffID, error
	UpdateStaffMember(ctx context.Context, tenant, staffID, role string) error
	RemoveStaffMember(ctx context.Context, tenant, staffID string) error
	ListStaffMembers(ctx context.Context, tenant, studioID string) ([]*StaffMember, error)

	// Studio users management
	AddStudioUser(ctx context.Context, tenant, studioID, email, password, role, displayName, username, firstName, lastName string) (string, error) // userID, error
	UpdateStudioUser(ctx context.Context, tenant, userID, studioID, email, role, displayName, username, firstName, lastName string, updateMask *fieldmaskpb.FieldMask) error
	RemoveStudioUser(ctx context.Context, tenant, userID, studioID string) error

	// Permissions
	SetStaffPermissions(ctx context.Context, tenant, staffID string, permissions []string) error
	GetStaffPermissions(ctx context.Context, tenant, staffID string) ([]string, error)
}

type UserRepository interface {
	ChangePassword(ctx context.Context, tenant, email, oldPassword, newPassword string) error
	ChangeEmail(ctx context.Context, tenant, email, password, newEmail string) error
	GetUserByEmail(ctx context.Context, tenant, email string) (string, string, string, error)
	GetUserByID(ctx context.Context, tenant, userID string) (*User, error)
	GetAllUsers(ctx context.Context, tenant string) ([]*User, error)
	UpdateUser(ctx context.Context, tenant string, user *User) error
	InsertUser(ctx context.Context, tenant string, user *User) error
	DeleteUser(ctx context.Context, tenant, userID string) error
}
