package usersevice

import (
	"context"
	"errors"
	"time"

	"userservice/internal/domain/models"
	"userservice/internal/lib/metrics"
	"userservice/internal/middleware"
	"userservice/internal/services/userservice"

	userservicev1 "github.com/MarkovMaksim2/protos/gen/go/userservice"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UserService interface {
	GetUser(
		ctx context.Context,
		userID int64) (*models.User, error)
	UpdateUser(
		ctx context.Context,
		user *models.User) (*models.User, error)
}

type serverAPI struct {
	userservicev1.UnimplementedUserServiceServer
	userService UserService
}

func Register(gRPC *grpc.Server, userService UserService) {
	userservicev1.RegisterUserServiceServer(gRPC, &serverAPI{userService: userService})
}

func (UserService *serverAPI) GetUser(
	ctx context.Context,
	req *userservicev1.GetUserRequest) (*userservicev1.GetUserResponse, error) {
	start := time.Now()
	var exitCode int = int(codes.OK)
	defer func() {
		metrics.ObserveRequest("GetUser", exitCode, time.Since(start))
	}()
	user, err := UserService.userService.GetUser(ctx, req.GetUserId())

	if err != nil {
		if errors.Is(err, userservice.ErrUserNotFound) {
			exitCode = int(codes.NotFound)
			return nil, status.Error(codes.NotFound, "user not found")
		}
		exitCode = int(codes.Internal)
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &userservicev1.GetUserResponse{
		User: &userservicev1.User{
			Id:      user.ID,
			Name:    user.Name,
			Surname: user.Surname,
			Avatar:  user.Avatar,
		},
	}, nil
}

func (UserService *serverAPI) UpdateUser(
	ctx context.Context,
	req *userservicev1.UpdateUserRequest) (*userservicev1.UpdateUserResponse, error) {
	start := time.Now()
	var exitCode int = int(codes.OK)
	defer func() {
		metrics.ObserveRequest("UpdateUser", exitCode, time.Since(start))
	}()
	userId := ctx.Value(middleware.UserIDKey)
	email := ctx.Value(middleware.EmailKey)

	if userId == nil || email == nil {
		exitCode = int(codes.Unauthenticated)
		return nil, status.Error(codes.Unauthenticated, "unauthenticated")
	}

	user := &models.User{
		Name:    req.GetUser().GetName(),
		Surname: req.GetUser().GetSurname(),
		Avatar:  req.GetUser().GetAvatar(),
	}

	user.ID = userId.(int64)
	updatedUser, err := UserService.userService.UpdateUser(ctx, user)
	if err != nil {
		if errors.Is(err, userservice.ErrUserNotFound) {
			exitCode = int(codes.Unauthenticated)
			return nil, status.Error(codes.NotFound, "user not found")
		}
		exitCode = int(codes.Internal)
		if errors.Is(err, userservice.ErrInvalidUserName) {
			return nil, status.Error(codes.Internal, "invalid user name")
		} else if errors.Is(err, userservice.ErrInvalidUserSurname) {
			return nil, status.Error(codes.Internal, "invalid user surname")
		} else if errors.Is(err, userservice.ErrInvalidUserAvatar) {
			return nil, status.Error(codes.Internal, "invalid user avatar")
		}

		return nil, status.Error(codes.Internal, "internal error")
	}

	return &userservicev1.UpdateUserResponse{
		User: &userservicev1.User{
			Id:      updatedUser.ID,
			Name:    updatedUser.Name,
			Surname: updatedUser.Surname,
			Avatar:  updatedUser.Avatar,
		},
	}, nil
}
