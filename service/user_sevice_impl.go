package service

import (
	"context"
	"donation/entity/client"
	"donation/entity/domain"
	"donation/exception"
	"donation/helper.go"
	"donation/repository"
	"errors"
	"github.com/go-redis/redis/v9"
	mail "github.com/xhit/go-simple-mail/v2"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
)

type UserServiceImpl struct {
	UserRepository repository.UserRepository
	DB             *gorm.DB
	Validate       *validator.Validate
	Chache         *redis.Client
	Smtp           *mail.SMTPClient
}

func NewUserService(userRepository repository.UserRepository, chc *redis.Client, DB *gorm.DB, validate *validator.Validate, smtp *mail.SMTPClient) UserService {
	return &UserServiceImpl{
		UserRepository: userRepository,
		DB:             DB,
		Validate:       validate,
		Chache:         chc,
		Smtp:           smtp,
	}
}

func (service *UserServiceImpl) Create(ctx context.Context, request client.UserCreateRequest) client.UserResponse {
	err := service.Validate.Struct(request)
	helper.PanicIfError(err)

	tx := service.DB.Begin()
	defer helper.CommitOrRollback(tx)

	userEmail, err := service.UserRepository.FindByEmail(ctx, service.Chache, tx, request.Email)
	helper.PanicIfError(err)
	exception.PanicIfEmailUsed(request.Email, userEmail.Email)

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.MinCost)
	helper.PanicIfError(err)

	goodEmail := strings.ToLower(request.Email)
	user := domain.User{
		FirstName:    request.FirstName,
		LastName:     request.LastName,
		Email:        goodEmail,
		Bio:          request.Bio,
		PasswordHash: string(passwordHash),
		IsActive:     false,
	}

	stringOtp := helper.GenerateOtp()

	otp := domain.OTP{
		Email: goodEmail,
		OTP:   stringOtp,
	}

	go helper.SendOtp(otp, service.Smtp)
	newUser := service.UserRepository.Save(ctx, service.Chache, tx, user, otp)

	return helper.ToUserResponse(newUser)
}

func (service *UserServiceImpl) Update(ctx context.Context, request client.UserUpdateRequest) client.UserResponse {
	err := service.Validate.Struct(request)
	helper.PanicIfError(err)

	tx := service.DB.Begin()
	defer helper.CommitOrRollback(tx)

	user, err := service.UserRepository.FindById(ctx, service.Chache, tx, request.Id)
	helper.PanicIfError(err)
	exception.PanicIfNotFound(user.Id)

	goodEmail := strings.ToLower(request.Email)

	if user.Email != goodEmail {
		userEmail, err := service.UserRepository.FindByEmail(ctx, service.Chache, tx, goodEmail)
		helper.PanicIfError(err)
		exception.PanicIfEmailUsed(goodEmail, userEmail.Email)
	}

	user.FirstName = request.FirstName
	user.LastName = request.LastName
	user.Email = goodEmail
	user.Bio = request.Bio

	updatedUser := service.UserRepository.Update(ctx, service.Chache, tx, user)

	return helper.ToUserResponse(updatedUser)
}

func (service *UserServiceImpl) Delete(ctx context.Context, userId int) {
	tx := service.DB.Begin()
	defer helper.CommitOrRollback(tx)

	user, err := service.UserRepository.FindById(ctx, service.Chache, tx, userId)
	helper.PanicIfError(err)
	exception.PanicIfNotFound(user.Id)

	service.UserRepository.Delete(ctx, service.Chache, tx, user)

}

func (service *UserServiceImpl) Session(ctx context.Context, request client.UserSessionRequest) client.UserResponse {
	err := service.Validate.Struct(request)
	helper.PanicIfError(err)

	tx := service.DB.Begin()
	defer helper.CommitOrRollback(tx)

	user, err := service.UserRepository.FindByEmail(ctx, service.Chache, tx, request.Email)
	helper.PanicIfError(err)
	exception.PanicIfNotFound(user.Id)

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(request.Password))
	if err != nil {
		panic(exception.NewWrongPasswordError(errors.New("wrong password")))
	}

	return helper.ToUserResponse(user)
}

func (service *UserServiceImpl) FindById(ctx context.Context, userId int) client.UserResponse {
	tx := service.DB.Begin()
	defer helper.CommitOrRollback(tx)

	user, err := service.UserRepository.FindById(ctx, service.Chache, tx, userId)
	helper.PanicIfError(err)
	exception.PanicIfNotFound(user.Id)

	return helper.ToUserResponse(user)
}

func (service *UserServiceImpl) FindByEmail(ctx context.Context, userEmail string) client.UserResponse {
	tx := service.DB.Begin()
	defer helper.CommitOrRollback(tx)

	user, err := service.UserRepository.FindByEmail(ctx, service.Chache, tx, userEmail)
	helper.PanicIfError(err)
	exception.PanicIfNotFound(user.Id)

	return helper.ToUserResponse(user)
}

func (service *UserServiceImpl) FindAll(ctx context.Context) []client.UserResponse {
	tx := service.DB.Begin()
	defer helper.CommitOrRollback(tx)

	users := service.UserRepository.FindAll(ctx, tx)

	return helper.ToUserResponses(users)
}

func (service *UserServiceImpl) FindOtp(ctx context.Context, request client.UserOtpRequest) client.UserResponse {
	err := service.Validate.Struct(request)
	helper.PanicIfError(err)

	tx := service.DB.Begin()
	defer helper.CommitOrRollback(tx)

	userOtp := domain.OTP{
		Email: request.Email,
		OTP:   request.OTP,
	}

	otp, err := service.UserRepository.FindOTp(ctx, service.Chache, userOtp)
	if err == redis.Nil {
		panic(exception.NewNotFoundError(errors.New("otp not found")))
	}

	if request.OTP != otp.OTP {
		panic(exception.NewWrongOtpError(errors.New("otp invalid")))
	}

	user := service.UserRepository.UpdateStatusEmail(ctx, tx, otp)
	service.UserRepository.DelOTP(ctx, service.Chache, otp)

	return helper.ToUserResponse(user)
}
