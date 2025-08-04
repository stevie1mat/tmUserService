package controllers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"trademinutes-user/config"
	"trademinutes-user/middleware"
	"trademinutes-user/utils"

	"github.com/ElioCloud/shared-models/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GetProfileHandler returns the current user's profile
func GetProfileHandler(w http.ResponseWriter, r *http.Request) {
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

// GetProfileByIDHandler returns a user's profile by ID
func GetProfileByIDHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Extract user ID from URL
	userID := strings.TrimPrefix(r.URL.Path, "/api/profile/")
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

// UpdateProfileInfoHandler updates user profile information
func UpdateProfileInfoHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
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

	var req models.User
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding request body: %v\n", err)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	collection := config.GetDB().Collection("MyClusterCol")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{}

	// Update only non-zero values (if they are provided)
	if req.Program != "" {
		update["program"] = req.Program
	}
	if req.Location != "" {
		update["location"] = req.Location
	}
	if req.College != "" {
		update["college"] = req.College
	}
	if req.YearOfStudy != "" {
		update["yearOfStudy"] = req.YearOfStudy
	}
	if req.Bio != "" {
		update["bio"] = req.Bio
	}
	if len(req.Skills) > 0 {
		update["skills"] = req.Skills
	}
	if req.ProfilePictureURL != "" {
		update["profilePictureURL"] = req.ProfilePictureURL
	}
	if (req.Stats != models.ProfileStats{}) {
		update["stats"] = req.Stats
	}
	if len(req.Achievements) > 0 {
		update["achievements"] = req.Achievements
	}

	// Check if profile was previously incomplete
	var existingUser models.User
	err := collection.FindOne(ctx, bson.M{"email": email}).Decode(&existingUser)
	if err != nil {
		log.Printf("Failed to fetch existing user: %v\n", err)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	wasIncomplete := existingUser.College == "" || existingUser.Program == "" || existingUser.YearOfStudy == ""
	isNowComplete := req.College != "" && req.Program != "" && req.YearOfStudy != ""

	// If profile is being completed for the first time, set credits to 200 (not increment)
	if wasIncomplete && isNowComplete {
		update["credits"] = 200
	}

	if len(update) == 0 {
		http.Error(w, "No valid fields to update", http.StatusBadRequest)
		return
	}

	result, err := collection.UpdateOne(
		ctx,
		bson.M{"email": email},
		bson.M{"$set": update},
	)

	if err != nil {
		log.Printf("Failed to update profile: %v\n", err)
		http.Error(w, "Failed to update profile", http.StatusInternalServerError)
		return
	}

	if result.MatchedCount == 0 {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Profile updated successfully",
	})
}

// UploadImageHandler handles profile picture uploads
func UploadImageHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Entered UploadImageHandler")

	email, ok := r.Context().Value(middleware.EmailKey).(string)
	if !ok {
		log.Println("Failed to get email from context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse multipart form (max 10MB)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		log.Printf("Error parsing multipart form: %v\n", err)
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Get the file from the form
	file, header, err := r.FormFile("image")
	if err != nil {
		log.Printf("Error getting file from form: %v\n", err)
		http.Error(w, "No image file provided", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Validate file type
	ext := strings.ToLower(filepath.Ext(header.Filename))
	allowedExts := []string{".jpg", ".jpeg", ".png", ".gif", ".webp"}
	allowed := false
	for _, allowedExt := range allowedExts {
		if ext == allowedExt {
			allowed = true
			break
		}
	}
	if !allowed {
		http.Error(w, "Invalid file type. Only JPG, PNG, GIF, and WebP are allowed", http.StatusBadRequest)
		return
	}

	// Validate file size (max 5MB)
	if header.Size > 5<<20 {
		http.Error(w, "File too large. Maximum size is 5MB", http.StatusBadRequest)
		return
	}

	// Read file content
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		log.Printf("Error reading file: %v\n", err)
		http.Error(w, "Failed to read file", http.StatusInternalServerError)
		return
	}

	// Try to upload to Cloudinary first
	var imageData string
	cloudinaryURL, err := utils.UploadProfileImageToCloudinary(fileBytes, header.Header.Get("Content-Type"), email)
	if err == nil {
		// Successfully uploaded to Cloudinary
		imageData = cloudinaryURL
		log.Printf("✅ Profile image uploaded to Cloudinary for user %s", email)
	} else {
		// Fallback to base64 storage
		log.Printf("⚠️  Cloudinary upload failed for user %s: %v, falling back to base64", email, err)
		base64Data := base64.StdEncoding.EncodeToString(fileBytes)
		imageData = "data:" + header.Header.Get("Content-Type") + ";base64," + base64Data
	}

	// Get the current user to check if they have an existing profile picture
	collection := config.GetDB().Collection("MyClusterCol")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var existingUser models.User
	err = collection.FindOne(ctx, bson.M{"email": email}).Decode(&existingUser)
	if err != nil {
		log.Printf("Failed to fetch existing user: %v", err)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Store the old profile picture URL for cleanup
	oldProfilePictureURL := existingUser.ProfilePictureURL

	// Update user's profile picture URL in database
	result, err := collection.UpdateOne(
		ctx,
		bson.M{"email": email},
		bson.M{"$set": bson.M{"profilePictureURL": imageData}},
	)
	if err != nil {
		log.Printf("Failed to update profile picture: %v\n", err)
		http.Error(w, "Failed to save profile picture", http.StatusInternalServerError)
		return
	}

	if result.MatchedCount == 0 {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Clean up old profile picture if it exists and is different
	if oldProfilePictureURL != "" && oldProfilePictureURL != imageData {
		go cleanupOldProfilePicture(oldProfilePictureURL)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Profile picture uploaded successfully",
		"url":     imageData,
	})
}

// UploadCoverImageHandler handles cover image uploads
func UploadCoverImageHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Entered UploadCoverImageHandler")

	email, ok := r.Context().Value(middleware.EmailKey).(string)
	if !ok {
		log.Println("Failed to get email from context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse multipart form (max 10MB)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		log.Printf("Error parsing multipart form: %v\n", err)
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Get the file from the form
	file, header, err := r.FormFile("image")
	if err != nil {
		log.Printf("Error getting file from form: %v\n", err)
		http.Error(w, "No image file provided", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Validate file type
	ext := strings.ToLower(filepath.Ext(header.Filename))
	allowedExts := []string{".jpg", ".jpeg", ".png", ".gif", ".webp"}
	allowed := false
	for _, allowedExt := range allowedExts {
		if ext == allowedExt {
			allowed = true
			break
		}
	}
	if !allowed {
		http.Error(w, "Invalid file type. Only JPG, PNG, GIF, and WebP are allowed", http.StatusBadRequest)
		return
	}

	// Validate file size (max 5MB)
	if header.Size > 5<<20 {
		http.Error(w, "File too large. Maximum size is 5MB", http.StatusBadRequest)
		return
	}

	// Read file content
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		log.Printf("Error reading file: %v\n", err)
		http.Error(w, "Failed to read file", http.StatusInternalServerError)
		return
	}

	// Try to upload to Cloudinary first
	var imageData string
	cloudinaryURL, err := utils.UploadCoverImageToCloudinary(fileBytes, header.Header.Get("Content-Type"), email)
	if err == nil {
		// Successfully uploaded to Cloudinary
		imageData = cloudinaryURL
		log.Printf("✅ Cover image uploaded to Cloudinary for user %s", email)
	} else {
		// Fallback to base64 storage
		log.Printf("⚠️  Cloudinary upload failed for user %s: %v, falling back to base64", email, err)
		base64Data := base64.StdEncoding.EncodeToString(fileBytes)
		imageData = "data:" + header.Header.Get("Content-Type") + ";base64," + base64Data
	}

	// Get the current user to check if they have an existing cover image
	collection := config.GetDB().Collection("MyClusterCol")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var existingUser models.User
	err = collection.FindOne(ctx, bson.M{"email": email}).Decode(&existingUser)
	if err != nil {
		log.Printf("Failed to fetch existing user: %v", err)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Store the old cover image URL for cleanup
	oldCoverImageURL := existingUser.CoverImageURL

	// Update user's cover image URL in database
	result, err := collection.UpdateOne(
		ctx,
		bson.M{"email": email},
		bson.M{"$set": bson.M{"coverImageURL": imageData}},
	)
	if err != nil {
		log.Printf("Failed to update cover image: %v\n", err)
		http.Error(w, "Failed to save cover image", http.StatusInternalServerError)
		return
	}

	if result.MatchedCount == 0 {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Clean up old cover image if it exists and is different
	if oldCoverImageURL != "" && oldCoverImageURL != imageData {
		go cleanupOldCoverImage(oldCoverImageURL)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Cover image uploaded successfully",
		"url":     imageData,
	})
}

// Helper functions for cleanup
func cleanupOldProfilePicture(oldProfilePictureURL string) error {
	// Check if it's a Cloudinary URL
	if strings.Contains(oldProfilePictureURL, "cloudinary.com") {
		publicID := utils.ExtractPublicIDFromURL(oldProfilePictureURL)
		if publicID != "" {
			if err := utils.DeleteImageFromCloudinary(publicID); err != nil {
				log.Printf("Failed to delete old profile picture from Cloudinary: %v", err)
				return err
			}
			log.Printf("✅ Successfully deleted old profile picture from Cloudinary: %s", publicID)
		}
	} else {
		// For base64 data, just log the cleanup
		log.Printf("Cleaning up old profile picture (base64): %s", oldProfilePictureURL[:50]+"...")
	}

	return nil
}

func cleanupOldCoverImage(oldCoverImageURL string) error {
	// Check if it's a Cloudinary URL
	if strings.Contains(oldCoverImageURL, "cloudinary.com") {
		publicID := utils.ExtractPublicIDFromURL(oldCoverImageURL)
		if publicID != "" {
			if err := utils.DeleteImageFromCloudinary(publicID); err != nil {
				log.Printf("Failed to delete old cover image from Cloudinary: %v", err)
				return err
			}
			log.Printf("✅ Successfully deleted old cover image from Cloudinary: %s", publicID)
		}
	} else {
		// For base64 data, just log the cleanup
		log.Printf("Cleaning up old cover image (base64): %s", oldCoverImageURL[:50]+"...")
	}

	return nil
}
