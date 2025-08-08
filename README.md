# MDP Project Backend

A Go-based backend API using Fiber framework with MongoDB for user authentication and session management.

## Features

- User authentication with JWT tokens
- Password validation and change functionality
- Role-based access control (Admin, Manager, User)
- Activity logging
- Input validation
- Password hashing with bcrypt
- CORS support

## Prerequisites

- Go 1.24.5 or later
- MongoDB (local or remote)
- Git

## Setup Instructions

### 1. Clone and Navigate
```bash
cd mdp-project-backend
```

### 2. Install Dependencies
```bash
go mod tidy
```

### 3. Configure Database
Edit `config/database.go` to set your MongoDB connection string:
```go
mongoURI := "mongodb://localhost:27017" // Change this to your MongoDB URI
```

### 4. Create Test Users
Run the seed script to create test users:
```bash
go run seed.go
```

### 5. Run the Server
```bash
go run main.go
```

The server will start on port 3033.

## Test Credentials

After running the seed script, you can use these credentials:

- **Admin**: username=`admin`, password=`password123!`
- **Manager**: username=`manager`, password=`password123!`  
- **User**: username=`user`, password=`password123!`

## API Endpoints

### Public Routes
- `GET /` - API information
- `POST /api/auth/login` - User login

### Protected Routes (Requires Bearer Token)
- `GET /api/profile` - Get user profile
- `POST /api/change-password` - Change password
- `POST /api/logout` - Logout (logs activity)

## Request Examples

### Login
```bash
curl -X POST http://localhost:3033/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "password123!"}'
```

### Change Password
```bash
curl -X POST http://localhost:3033/api/change-password \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -d '{"old_password": "password123!", "new_password": "newPassword123!"}'
```

## Password Requirements

- At least 8 characters long
- Contains at least one letter
- Contains at least one number  
- Contains at least one special character

## Project Structure

```
mdp-project-backend/
├── main.go              # Main application entry point
├── seed.go              # Database seeding script
├── go.mod               # Go module dependencies
├── config/
│   └── database.go      # MongoDB connection
├── models/
│   └── user.go          # Data models
├── handlers/
│   └── auth.go          # HTTP handlers
├── middleware/
│   └── auth.go          # Authentication middleware
└── utils/
    └── auth.go          # Authentication utilities
```

## Security Features

- JWT token authentication
- Password hashing with bcrypt
- Input validation
- Rate limiting ready
- CORS enabled
- Activity logging

## Development

To modify the JWT secret key, update the `jwtSecret` variable in `utils/auth.go`:
```go
var jwtSecret = []byte("your-secret-key-change-this-in-production")
```
