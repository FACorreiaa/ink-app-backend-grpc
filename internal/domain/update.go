package domain

package studio

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"google.golang.org/genproto/protobuf/field_mask"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/yourorg/inkme-grpc/inkMe/studio"
)

type StudioRepository struct {
	pgpool *pgxpool.Pool
}

func NewStudioRepository(pgpool *pgxpool.Pool) *StudioRepository {
	return &StudioRepository{
		pgpool: pgpool,
	}
}

func (r *StudioRepository) UpdateStudio(ctx context.Context, req *pb.UpdateStudioRequest) (*pb.UpdateStudioResponse, error) {
	// Start a transaction
	tx, err := r.pgpool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to start transaction")
	}
	
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		}
	}()

	// First, get the current studio to apply updates
	currentStudio, err := r.getStudioByID(ctx, tx, req.StudioId)
	if err != nil {
		return nil, err
	}
	
	// Apply updates according to field mask
	updatedStudio, setClauses, args, err := applyFieldMaskUpdates(currentStudio, req.Studio, req.UpdateMask)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to apply updates: %v", err)
	}
	
	// If no fields to update, return the current studio
	if len(setClauses) == 0 {
		return &pb.UpdateStudioResponse{
			Base: &pb.BaseResponse{
				RequestId: req.Base.RequestId,
				Status: &pb.Status{
					Code:    pb.Status_OK,
					Message: "No fields to update",
				},
			},
			Studio: currentStudio,
		}, nil
	}
	
	// Build and execute the query
	query := "UPDATE studios SET " + strings.Join(setClauses, ", ") + " WHERE id = $" + fmt.Sprintf("%d", len(args)+1)
	args = append(args, req.StudioId)
	
	_, err = tx.Exec(ctx, query, args...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update studio: %v", err)
	}
	
	// Commit transaction
	if err = tx.Commit(ctx); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to commit transaction: %v", err)
	}
	
	return &pb.UpdateStudioResponse{
		Base: &pb.BaseResponse{
			RequestId: req.Base.RequestId,
			Status: &pb.Status{
				Code:    pb.Status_OK,
				Message: "Studio updated successfully",
			},
		},
		Studio: updatedStudio,
	}, nil
}

func (r *StudioRepository) getStudioByID(ctx context.Context, tx pgx.Tx, studioID string) (*pb.Studio, error) {
	query := `
		SELECT id, name, address, phone, email, website, created_at, updated_at
		FROM studios
		WHERE id = $1
	`
	
	row := tx.QueryRow(ctx, query, studioID)
	
	studio := &pb.Studio{}
	err := row.Scan(
		&studio.Id,
		&studio.Name,
		&studio.Address,
		&studio.Phone,
		&studio.Email,
		&studio.Website,
		&studio.CreatedAt,
		&studio.UpdatedAt,
	)
	
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, status.Errorf(codes.NotFound, "studio with ID %s not found", studioID)
		}
		return nil, status.Errorf(codes.Internal, "failed to get studio: %v", err)
	}
	
	return studio, nil
}

func applyFieldMaskUpdates(currentStudio *pb.Studio, updateStudio *pb.Studio, mask *field_mask.FieldMask) (*pb.Studio, []string, []interface{}, error) {
	if mask == nil || len(mask.Paths) == 0 {
		return nil, nil, nil, fmt.Errorf("update_mask is required")
	}
	
	result := &pb.Studio{
		Id:        currentStudio.Id,
		Name:      currentStudio.Name,
		Address:   currentStudio.Address,
		Phone:     currentStudio.Phone,
		Email:     currentStudio.Email,
		Website:   currentStudio.Website,
		CreatedAt: currentStudio.CreatedAt,
		UpdatedAt: currentStudio.UpdatedAt,
	}
	
	var setClauses []string
	var args []interface{}
	argIndex := 1
	
	// Process each field in the mask
	for _, path := range mask.Paths {
		switch path {
		case "name":
			if updateStudio.Name != "" {
				result.Name = updateStudio.Name
				setClauses = append(setClauses, fmt.Sprintf("name = $%d", argIndex))
				args = append(args, updateStudio.Name)
				argIndex++
			}
		case "address":
			if updateStudio.Address != "" {
				result.Address = updateStudio.Address
				setClauses = append(setClauses, fmt.Sprintf("address = $%d", argIndex))
				args = append(args, updateStudio.Address)
				argIndex++
			}
		case "phone":
			if updateStudio.Phone != "" {
				result.Phone = updateStudio.Phone
				setClauses = append(setClauses, fmt.Sprintf("phone = $%d", argIndex))
				args = append(args, updateStudio.Phone)
				argIndex++
			}
		case "email":
			if updateStudio.Email != "" {
				result.Email = updateStudio.Email
				setClauses = append(setClauses, fmt.Sprintf("email = $%d", argIndex))
				args = append(args, updateStudio.Email)
				argIndex++
			}
		case "website":
			if updateStudio.Website != "" {
				result.Website = updateStudio.Website
				setClauses = append(setClauses, fmt.Sprintf("website = $%d", argIndex))
				args = append(args, updateStudio.Website)
				argIndex++
			}
		default:
			return nil, nil, nil, fmt.Errorf("unknown field in update_mask: %s", path)
		}
	}
	
	// Always update the updated_at timestamp
	setClauses = append(setClauses, fmt.Sprintf("updated_at = $%d", argIndex))
	now := time.Now().Unix()
	args = append(args, now)
	result.UpdatedAt = now
	
	return result, setClauses, args, nil
}