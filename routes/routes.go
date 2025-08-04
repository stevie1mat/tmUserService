package routes

import (
	"net/http"
	"trademinutes-user/controllers"
	"trademinutes-user/middleware"

	"github.com/gorilla/mux"
)

func SetupRoutes(router *mux.Router) {
	// Auth routes
	authRouter := router.PathPrefix("/api/auth").Subrouter()
	authRouter.HandleFunc("/register", controllers.RegisterHandler).Methods("POST", "OPTIONS")
	authRouter.HandleFunc("/login", controllers.LoginHandler).Methods("POST", "OPTIONS")
	authRouter.Handle("/profile", middleware.JWTAuthMiddleware(http.HandlerFunc(controllers.ProfileHandler))).Methods("GET", "OPTIONS")
	authRouter.HandleFunc("/user/{id}", controllers.GetUserByIDHandler).Methods("GET", "OPTIONS")
	authRouter.HandleFunc("/users", controllers.GetAllUsersHandler).Methods("GET", "OPTIONS")
	authRouter.HandleFunc("/admin/delete/{id}", controllers.AdminDeleteUserHandler).Methods("DELETE", "OPTIONS")
	authRouter.HandleFunc("/update-credits", controllers.UpdateCreditsHandler).Methods("PUT", "OPTIONS")

	// Profile routes (protected)
	profileRouter := router.PathPrefix("/api/profile").Subrouter()
	profileRouter.Use(middleware.JWTMiddleware)
	profileRouter.HandleFunc("/get", controllers.GetProfileHandler).Methods("GET", "OPTIONS")
	profileRouter.HandleFunc("/{userId}", controllers.GetProfileByIDHandler).Methods("GET", "OPTIONS")
	profileRouter.HandleFunc("/update-info", controllers.UpdateProfileInfoHandler).Methods("POST", "OPTIONS")
	profileRouter.HandleFunc("/upload-image", controllers.UploadImageHandler).Methods("POST", "OPTIONS")
	profileRouter.HandleFunc("/upload-cover-image", controllers.UploadCoverImageHandler).Methods("POST", "OPTIONS")

	// Public admin routes (for admin dashboard)
	router.HandleFunc("/api/users", controllers.GetAllUsersHandler).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/admin/delete/{id}", controllers.AdminDeleteUserHandler).Methods("DELETE", "OPTIONS")
}
