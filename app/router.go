package app

import (
	"donation/exception"
	"donation/handler"

	"github.com/julienschmidt/httprouter"
)

func NewRouter(userHandler handler.UserHandler) *httprouter.Router {
	router := httprouter.New()

	router.GET("/api/users", userHandler.FindAll)
	router.GET("/api/users/:userId", userHandler.FindById)
	// router.GET("/api/users/email/:userEmail", userHandler.FindByEmail)
	router.POST("/api/users", userHandler.Create)
	router.PUT("/api/users", userHandler.Update)
	router.DELETE("/api/users", userHandler.Delete)
	router.POST("/api/users/session", userHandler.Session)
	router.POST("/api/users/otp", userHandler.FindOtp)
	router.POST("/api/users/otp/request-otp", userHandler.GetNewOtp)

	router.PanicHandler = exception.ErrorHandler

	return router
}
