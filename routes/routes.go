package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"trademinutes-user/config"
	"trademinutes-user/controllers"
	"trademinutes-user/middleware"

	"github.com/ElioCloud/shared-models/models"
	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
)

func SetupRoutes(router *mux.Router) {
	// Auth routes
	authRouter := router.PathPrefix("/api/auth").Subrouter()
	authRouter.HandleFunc("/register", controllers.RegisterHandler).Methods("POST", "OPTIONS")
	authRouter.HandleFunc("/login", controllers.LoginHandler).Methods("POST", "OPTIONS")
	authRouter.HandleFunc("/github", controllers.OAuthHandler).Methods("POST", "OPTIONS")
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

	// Public endpoint for admin page (no authentication required)
	router.HandleFunc("/api/admin/users", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Get search query parameter
		searchQuery := r.URL.Query().Get("q")

		collection := config.GetDB().Collection("MyClusterCol")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		var filter bson.M
		if searchQuery != "" {
			// Search by name or email (case-insensitive)
			filter = bson.M{
				"$or": []bson.M{
					{"name": bson.M{"$regex": searchQuery, "$options": "i"}},
					{"email": bson.M{"$regex": searchQuery, "$options": "i"}},
				},
			}
		} else {
			filter = bson.M{}
		}

		fmt.Println("🔍 Executing database query with filter:", filter)
		cursor, err := collection.Find(ctx, filter)
		if err != nil {
			fmt.Println("❌ Database query error:", err)
			http.Error(w, "Failed to fetch users", http.StatusInternalServerError)
			return
		}
		fmt.Println("✅ Database query successful")
		defer cursor.Close(ctx)

		var users []models.User
		if err = cursor.All(ctx, &users); err != nil {
			http.Error(w, "Failed to decode users", http.StatusInternalServerError)
			return
		}

		// Remove passwords from response
		for i := range users {
			users[i].Password = ""
		}

		// Return in the format expected by the frontend
		response := map[string]interface{}{
			"data":  users,
			"count": len(users),
		}

		json.NewEncoder(w).Encode(response)
	}).Methods("GET", "OPTIONS")

	// Health check for admin
	router.HandleFunc("/api/admin/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "ok",
			"service":   "TradeMinutes User Service",
			"timestamp": time.Now().Unix(),
		})
	}).Methods("GET", "OPTIONS")

	// Test endpoint with mock data (no database required)
	router.HandleFunc("/api/admin/test-users", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(http.StatusOK)

		// Mock user data for testing
		mockUsers := []map[string]interface{}{
			{
				"_id":       "507f1f77bcf86cd799439011",
				"name":      "John Doe",
				"email":     "john@example.com",
				"college":   "MIT",
				"program":   "Computer Science",
				"credits":   150,
				"location":  "Boston, MA",
				"createdAt": 1640995200,
			},
			{
				"_id":       "507f1f77bcf86cd799439012",
				"name":      "Jane Smith",
				"email":     "jane@example.com",
				"college":   "Stanford",
				"program":   "Engineering",
				"credits":   200,
				"location":  "San Francisco, CA",
				"createdAt": 1640995200,
			},
			{
				"_id":       "507f1f77bcf86cd799439013",
				"name":      "Bob Johnson",
				"email":     "bob@example.com",
				"college":   "Harvard",
				"program":   "Business",
				"credits":   75,
				"location":  "Cambridge, MA",
				"createdAt": 1640995200,
			},
		}

		response := map[string]interface{}{
			"data":  mockUsers,
			"count": len(mockUsers),
		}

		json.NewEncoder(w).Encode(response)
	}).Methods("GET", "OPTIONS")
}
