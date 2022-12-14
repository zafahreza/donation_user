package main

import (
	"donation/app"
	"donation/chache"
	"donation/handler"
	"donation/helper"
	"donation/middleware"
	"donation/repository"
	"donation/service"
	"fmt"
	"net/http"

	"github.com/go-playground/validator/v10"
)

func main() {
	db := app.NewSetupDB()
	validate := validator.New()
	chc := app.NewChacheDB()
	smtp := app.NewSmtpClient()

	userRepository := repository.NewUserRepository()
	userChache := chache.NewUserChache(chc)
	userService := service.NewUserService(userRepository, userChache, db, validate, smtp)
	authMiddleware := middleware.NewAuthMiddleware()
	userHandler := handler.NewUserHanlder(userService, authMiddleware)

	router := app.NewRouter(userHandler)

	server := http.Server{
		Addr:    "localhost:3333",
		Handler: router,
	}

	fmt.Println("server is running.....")

	err := server.ListenAndServe()
	helper.PanicIfError(err)
}
