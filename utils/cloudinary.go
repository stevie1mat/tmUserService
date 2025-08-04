package utils

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

var cld *cloudinary.Cloudinary

func InitCloudinary() error {
	cloudName := os.Getenv("CLOUDINARY_CLOUD_NAME")
	apiKey := os.Getenv("CLOUDINARY_API_KEY")
	apiSecret := os.Getenv("CLOUDINARY_API_SECRET")

	if cloudName == "" || apiKey == "" || apiSecret == "" {
		return fmt.Errorf("Cloudinary configuration missing. Please set CLOUDINARY_CLOUD_NAME, CLOUDINARY_API_KEY, and CLOUDINARY_API_SECRET")
	}

	var err error
	cld, err = cloudinary.NewFromParams(cloudName, apiKey, apiSecret)
	if err != nil {
		return fmt.Errorf("failed to initialize Cloudinary: %v", err)
	}

	log.Println("✅ Cloudinary initialized successfully")
	return nil
}

func UploadProfileImageToCloudinary(imageData []byte, contentType, email string) (string, error) {
	if cld == nil {
		return "", fmt.Errorf("Cloudinary not initialized")
	}

	// Generate unique filename
	timestamp := time.Now().Unix()
	filename := fmt.Sprintf("profile_%s_%d", email, timestamp)

	// Get file extension
	ext := getExtensionFromContentType(contentType)
	fullFilename := filename + ext

	// Upload to Cloudinary with face detection and specific dimensions
	ctx := context.Background()
	result, err := cld.Upload.Upload(ctx, imageData, uploader.UploadParams{
		PublicID:       fmt.Sprintf("trademinutes/profiles/%s", fullFilename),
		ResourceType:   "image",
		Transformation: "f_auto,q_auto,w_400,h_400,c_fill,g_face",
		Overwrite:      &[]bool{false}[0],
	})

	if err != nil {
		return "", fmt.Errorf("failed to upload to Cloudinary: %v", err)
	}

	return result.SecureURL, nil
}

func UploadCoverImageToCloudinary(imageData []byte, contentType, email string) (string, error) {
	if cld == nil {
		return "", fmt.Errorf("Cloudinary not initialized")
	}

	// Generate unique filename
	timestamp := time.Now().Unix()
	filename := fmt.Sprintf("cover_%s_%d", email, timestamp)

	// Get file extension
	ext := getExtensionFromContentType(contentType)
	fullFilename := filename + ext

	// Upload to Cloudinary with specific dimensions for cover images
	ctx := context.Background()
	result, err := cld.Upload.Upload(ctx, imageData, uploader.UploadParams{
		PublicID:       fmt.Sprintf("trademinutes/covers/%s", fullFilename),
		ResourceType:   "image",
		Transformation: "f_auto,q_auto,w_1200,h_400,c_fill",
		Overwrite:      &[]bool{false}[0],
	})

	if err != nil {
		return "", fmt.Errorf("failed to upload to Cloudinary: %v", err)
	}

	return result.SecureURL, nil
}

func UploadBase64ImageToCloudinary(base64Data, contentType, email string) (string, error) {
	// Remove data URL prefix if present
	if strings.HasPrefix(base64Data, "data:") {
		parts := strings.Split(base64Data, ",")
		if len(parts) != 2 {
			return "", fmt.Errorf("invalid base64 data URL format")
		}
		base64Data = parts[1]
	}

	// Decode base64 data
	imageData, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 data: %v", err)
	}

	// Upload to Cloudinary
	return UploadProfileImageToCloudinary(imageData, contentType, email)
}

func DeleteImageFromCloudinary(publicID string) error {
	if cld == nil {
		return fmt.Errorf("Cloudinary not initialized")
	}

	ctx := context.Background()
	_, err := cld.Upload.Destroy(ctx, uploader.DestroyParams{
		PublicID: publicID,
	})

	if err != nil {
		return fmt.Errorf("failed to delete from Cloudinary: %v", err)
	}

	return nil
}

func ExtractPublicIDFromURL(url string) string {
	// Extract public ID from Cloudinary URL
	// Example: https://res.cloudinary.com/cloud_name/image/upload/v1234567890/folder/filename.jpg
	// We want: folder/filename

	parts := strings.Split(url, "/")
	if len(parts) < 2 {
		return ""
	}

	// Find the "upload" part
	uploadIndex := -1
	for i, part := range parts {
		if part == "upload" {
			uploadIndex = i
			break
		}
	}

	if uploadIndex == -1 || uploadIndex+2 >= len(parts) {
		return ""
	}

	// Get the part after "upload" and before the version
	pathParts := parts[uploadIndex+2:]

	// Remove version if present
	if len(pathParts) > 0 && strings.HasPrefix(pathParts[0], "v") {
		pathParts = pathParts[1:]
	}

	// Join the remaining parts
	return strings.Join(pathParts, "/")
}

func getExtensionFromContentType(contentType string) string {
	switch contentType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	default:
		return ".jpg" // Default to jpg
	}
}
