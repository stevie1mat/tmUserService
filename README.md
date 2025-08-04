# TradeMinutes User Service

A unified Go service that combines authentication, user management, and profile functionality for the TradeMinutes platform.

## 🚀 Features

### Authentication
- User registration and login
- JWT token generation and validation
- Password hashing with bcrypt
- OAuth support (GitHub, Google)

### User Management
- Get user by ID
- Get all users (admin)
- Update user credits
- Delete users (admin)

### Profile Management
- Get and update user profiles
- Profile picture upload with Cloudinary integration
- Cover image upload with Cloudinary integration
- Profile completion tracking

### Image Upload
- Cloudinary CDN integration for profile and cover images
- Automatic image optimization and face detection
- Fallback to base64 storage if Cloudinary fails
- Automatic cleanup of old images

## 📁 Project Structure

```
TradeMinutesUserService/
├── config/
│   └── config.go          # Database configuration
├── controllers/
│   ├── auth.go            # Authentication controllers
│   └── profile.go         # Profile management controllers
├── middleware/
│   └── auth_middleware.go # JWT authentication middleware
├── routes/
│   └── routes.go          # Route definitions
├── utils/
│   └── cloudinary.go      # Cloudinary utility functions
├── main.go                # Application entry point
├── go.mod                 # Go module dependencies
└── README.md             # This file
```

## 🔧 Setup

### Prerequisites
- Go 1.21 or higher
- MongoDB database
- Cloudinary account (optional, for image uploads)

### Environment Variables

Create a `.env` file in the root directory:

```env
# Database
MONGO_URI=mongodb://localhost:27017
DB_NAME=trademinutes

# JWT
JWT_SECRET=your-secret-key-here

# Cloudinary (optional)
CLOUDINARY_CLOUD_NAME=your-cloud-name
CLOUDINARY_API_KEY=your-api-key
CLOUDINARY_API_SECRET=your-api-secret

# Server
PORT=8080
ENV=development
```

### Installation

1. Clone the repository
2. Navigate to the service directory:
   ```bash
   cd TradeMinutesUserService
   ```

3. Install dependencies:
   ```bash
   go mod tidy
   ```

4. Run the service:
   ```bash
   go run main.go
   ```

The service will start on port 8080 by default.

## 📡 API Endpoints

### Authentication
- `POST /api/auth/register` - Register a new user
- `POST /api/auth/login` - Login user
- `GET /api/auth/profile` - Get current user profile (protected)

### User Management
- `GET /api/auth/user/{id}` - Get user by ID
- `GET /api/auth/users` - Get all users (admin)
- `PUT /api/auth/update-credits` - Update user credits (protected)
- `DELETE /api/auth/admin/delete/{id}` - Delete user (admin)

### Profile Management
- `GET /api/profile/get` - Get current user profile (protected)
- `GET /api/profile/{userId}` - Get user profile by ID (protected)
- `POST /api/profile/update-info` - Update profile information (protected)
- `POST /api/profile/upload-image` - Upload profile picture (protected)
- `POST /api/profile/upload-cover-image` - Upload cover image (protected)

### Public Admin Endpoints
- `GET /api/users` - Get all users (public, for admin dashboard)
- `DELETE /api/admin/delete/{id}` - Delete user (public, for admin dashboard)

## 🔐 Authentication

The service uses JWT tokens for authentication. Protected endpoints require a valid JWT token in the Authorization header:

```
Authorization: Bearer <your-jwt-token>
```

## 🖼️ Image Upload

### Profile Pictures
- Supported formats: JPG, PNG, GIF, WebP
- Maximum size: 5MB
- Cloudinary optimization: 400x400 with face detection
- Automatic cleanup of old images

### Cover Images
- Supported formats: JPG, PNG, GIF, WebP
- Maximum size: 5MB
- Cloudinary optimization: 1200x400
- Automatic cleanup of old images

## 🧪 Testing

Test the service endpoints:

```bash
# Health check
curl http://localhost:8080/ping

# Register a user
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123","name":"Test User"}'

# Login
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123"}'
```

## 🔄 Migration from Separate Services

This service consolidates functionality from:
- `TradeMinutesAuth` (port 8080)
- `TradeMinutesProfile` (port 8081)

### Frontend Changes Required

Update your frontend API calls to use the new unified service:

```typescript
// Old: Separate services
const authUrl = 'http://localhost:8080/api/auth';
const profileUrl = 'http://localhost:8081/api/profile';

// New: Unified service
const userServiceUrl = 'http://localhost:8080/api';
const authUrl = `${userServiceUrl}/auth`;
const profileUrl = `${userServiceUrl}/profile`;
```

## 🚀 Deployment

### Docker

Create a `Dockerfile`:

```dockerfile
FROM golang:1.21-alpine

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o main .

EXPOSE 8080

CMD ["./main"]
```

### Environment Variables

Make sure to set all required environment variables in your deployment environment.

## 📝 License

This project is part of the TradeMinutes platform. 