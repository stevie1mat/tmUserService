package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"trademinutes-user/config"
	"trademinutes-user/middleware"

	"github.com/ElioCloud/shared-models/models"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

// RegisterHandler handles user registration
func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	var user models.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if user.Email == "" || user.Password == "" || user.Name == "" {
		http.Error(w, "Email, password, and name are required", http.StatusBadRequest)
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}

	// Check if user already exists
	collection := config.GetDB().Collection("MyClusterCol")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var existingUser models.User
	err = collection.FindOne(ctx, bson.M{"email": strings.ToLower(user.Email)}).Decode(&existingUser)
	if err == nil {
		http.Error(w, "User already exists", http.StatusConflict)
		return
	}

	// Create new user
	user.ID = primitive.NewObjectID()
	user.Email = strings.ToLower(user.Email)
	user.Password = string(hashedPassword)
	user.Credits = 200 // Starting credits
	user.CreatedAt = time.Now().Unix()
	// user.Verified = false // Commented out as Verified field doesn't exist in models.User

	_, err = collection.InsertOne(ctx, user)
	if err != nil {
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	// Generate JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"email": user.Email,
		"exp":   time.Now().Add(time.Hour * 24 * 7).Unix(), // 7 days
	})

	tokenString, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	// Remove password from response
	user.Password = ""

	response := map[string]interface{}{
		"token": tokenString,
		"user":  user,
	}

	json.NewEncoder(w).Encode(response)
}

// LoginHandler handles user login
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	var loginRequest struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&loginRequest); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Find user
	collection := config.GetDB().Collection("MyClusterCol")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user models.User
	err := collection.FindOne(ctx, bson.M{"email": strings.ToLower(loginRequest.Email)}).Decode(&user)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Check password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(loginRequest.Password))
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Generate JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"email": user.Email,
		"exp":   time.Now().Add(time.Hour * 24 * 7).Unix(), // 7 days
	})

	tokenString, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	// Remove password from response
	user.Password = ""

	response := map[string]interface{}{
		"token": tokenString,
		"user":  user,
	}

	json.NewEncoder(w).Encode(response)
}

// ProfileHandler returns the current user's profile
func ProfileHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	email, ok := r.Context().Value(middleware.EmailKey).(string)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	collection := config.GetDB().Collection("MyClusterCol")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user models.User
	err := collection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Remove password from response
	user.Password = ""

	json.NewEncoder(w).Encode(user)
}

// GetUserByIDHandler returns a user by ID
func GetUserByIDHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Extract user ID from URL
	userID := strings.TrimPrefix(r.URL.Path, "/api/auth/user/")
	if userID == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	collection := config.GetDB().Collection("MyClusterCol")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user models.User
	err = collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&user)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Remove password from response
	user.Password = ""

	json.NewEncoder(w).Encode(user)
}

// GetAllUsersHandler returns all users (for admin)
func GetAllUsersHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Add debugging
	fmt.Println("🔍 GetAllUsersHandler called")
	fmt.Println("🔍 Request URL:", r.URL.String())

	// Get search query parameter
	searchQuery := r.URL.Query().Get("q")
	fmt.Println("🔍 Search query:", searchQuery)

	// Check if database is connected
	db := config.GetDB()
	if db == nil {
		fmt.Println("❌ Database is nil")
		http.Error(w, "Database connection failed", http.StatusInternalServerError)
		return
	}
	fmt.Println("✅ Database connected:", db.Name())

	collection := db.Collection("MyClusterCol")
	fmt.Println("✅ Collection accessed:", collection.Name())

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
	fmt.Println("🔍 Filter:", filter)

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		fmt.Println("❌ Database query error:", err)
		http.Error(w, "Failed to fetch users", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(ctx)

	var users []models.User
	if err = cursor.All(ctx, &users); err != nil {
		fmt.Println("❌ Cursor decode error:", err)
		http.Error(w, "Failed to decode users", http.StatusInternalServerError)
		return
	}

	fmt.Println("✅ Found", len(users), "users")

	// Remove passwords from response
	for i := range users {
		users[i].Password = ""
	}

	// Return in the format expected by the frontend
	response := map[string]interface{}{
		"data":  users,
		"count": len(users),
	}

	fmt.Println("✅ Sending response with", len(users), "users")
	json.NewEncoder(w).Encode(response)
}

// AdminDeleteUserHandler deletes a user (for admin)
func AdminDeleteUserHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Extract user ID from URL
	userID := strings.TrimPrefix(r.URL.Path, "/api/admin/delete/")
	if userID == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	collection := config.GetDB().Collection("MyClusterCol")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Delete user
	result, err := collection.DeleteOne(ctx, bson.M{"_id": objectID})
	if err != nil {
		http.Error(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}

	if result.DeletedCount == 0 {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// TODO: Cascade delete user's tasks, bookings, favorites, reviews
	// This would require connections to other services

	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "User deleted successfully",
	})
}

// UpdateCreditsHandler updates user credits
func UpdateCreditsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "PUT, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	email, ok := r.Context().Value(middleware.EmailKey).(string)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var request struct {
		Credits int `json:"credits"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	collection := config.GetDB().Collection("MyClusterCol")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := collection.UpdateOne(
		ctx,
		bson.M{"email": email},
		bson.M{"$set": bson.M{"credits": request.Credits}},
	)

	if err != nil {
		http.Error(w, "Failed to update credits", http.StatusInternalServerError)
		return
	}

	if result.MatchedCount == 0 {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Credits updated successfully",
		"credits": request.Credits,
	})
}

// DeductCreditsHandler deducts credits from user account
func DeductCreditsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	var request struct {
		UserId  string `json:"userId"`
		Credits int    `json:"credits"`
		Reason  string `json:"reason"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if request.Credits <= 0 {
		http.Error(w, "Credits to deduct must be positive", http.StatusBadRequest)
		return
	}

	collection := config.GetDB().Collection("MyClusterCol")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// First, get current user to check available credits
	var user models.User
	objectID, err := primitive.ObjectIDFromHex(request.UserId)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	err = collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&user)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	if user.Credits < request.Credits {
		http.Error(w, "Insufficient credits", http.StatusPaymentRequired)
		return
	}

	// Deduct credits
	result, err := collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$inc": bson.M{"credits": -request.Credits}},
	)

	if err != nil {
		http.Error(w, "Failed to deduct credits", http.StatusInternalServerError)
		return
	}

	if result.MatchedCount == 0 {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Get updated user to return remaining credits
	var updatedUser models.User
	err = collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&updatedUser)
	if err != nil {
		http.Error(w, "Failed to get updated user", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":           "Credits deducted successfully",
		"credits_deducted":  request.Credits,
		"remaining_credits": updatedUser.Credits,
		"reason":            request.Reason,
	})
}

// OAuthHandler handles OAuth user registration/login
func OAuthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	var oauthData struct {
		Email    string `json:"email"`
		Name     string `json:"name"`
		Provider string `json:"provider,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&oauthData); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if oauthData.Email == "" || oauthData.Name == "" {
		http.Error(w, "Email and name are required", http.StatusBadRequest)
		return
	}

	collection := config.GetDB().Collection("MyClusterCol")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check if user already exists
	var existingUser models.User
	err := collection.FindOne(ctx, bson.M{"email": strings.ToLower(oauthData.Email)}).Decode(&existingUser)

	if err == nil {
		// User exists, generate JWT token
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"email": existingUser.Email,
			"exp":   time.Now().Add(time.Hour * 24 * 7).Unix(), // 7 days
		})

		tokenString, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
		if err != nil {
			http.Error(w, "Failed to generate token", http.StatusInternalServerError)
			return
		}

		// Remove password from response
		existingUser.Password = ""

		response := map[string]interface{}{
			"token": tokenString,
			"user":  existingUser,
		}

		json.NewEncoder(w).Encode(response)
		return
	}

	// User doesn't exist, create new user
	user := models.User{
		ID:        primitive.NewObjectID(),
		Email:     strings.ToLower(oauthData.Email),
		Name:      oauthData.Name,
		Password:  "",  // OAuth users don't have passwords
		Credits:   200, // Starting credits
		CreatedAt: time.Now().Unix(),
	}

	_, err = collection.InsertOne(ctx, user)
	if err != nil {
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	// Generate JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"email": user.Email,
		"exp":   time.Now().Add(time.Hour * 24 * 7).Unix(), // 7 days
	})

	tokenString, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	// Remove password from response
	user.Password = ""

	response := map[string]interface{}{
		"token": tokenString,
		"user":  user,
	}

	json.NewEncoder(w).Encode(response)
}
