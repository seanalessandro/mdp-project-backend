# MDP Project - Complete Setup Guide

This project implements the authentication flow shown in the provided flowchart, with a Go backend (Fiber + MongoDB) and Next.js frontend (TypeScript + Tailwind + Ant Design).

## ✅ Implementation Status

### Backend (Go + Fiber + MongoDB)
- ✅ User authentication with JWT tokens
- ✅ Input validation (username/password format)
- ✅ Database credential verification
- ✅ Password complexity validation
- ✅ Role-based access control
- ✅ Password change functionality
- ✅ Activity logging
- ✅ Session management
- ✅ Logout functionality
- ✅ CORS middleware
- ✅ Error handling

### Frontend (Next.js + TypeScript + Tailwind + Ant Design)
- ✅ Login page with validation
- ✅ Role-based dashboard routing
- ✅ Protected routes
- ✅ Password change modal
- ✅ User session management
- ✅ JWT token handling
- ✅ Responsive design
- ✅ Input validation

## 🚀 Quick Start

### 1. Backend Setup

```bash
cd mdp-project-backend

# Install dependencies
go mod tidy

# Option A: With MongoDB (recommended)
# Start MongoDB service first, then:
go run seed.go  # Create test users
go run main.go  # Start server

# Option B: Demo mode (without MongoDB)
go run main_demo.go  # Start demo server
```

### 2. Frontend Setup

```bash
cd mdp-project-frontend/mdp-ss-xiii-mini-project

# Install dependencies (try one of these)
npm install --legacy-peer-deps
# or
yarn install

# Start development server
npm run dev
# or
yarn dev
```

## 🔐 Test Credentials

| Role    | Username | Password      |
|---------|----------|---------------|
| Admin   | admin    | password123!  |
| Manager | manager  | password123!  |
| User    | user     | password123!  |

## 📱 Application Flow (Matches Flowchart)

### Login Process
1. **Start** → User lands on login page
2. **Input Validation** → Username/password format check
3. **Credential Verification** → Database lookup and validation
4. **Role-based Routing** → Dashboard access based on user role

### Password Change Flow
1. **Current Password Verification** → Validates old password
2. **New Password Validation** → Checks complexity requirements
3. **Password Update** → Securely updates in database
4. **Activity Logging** → Records password change event

## 🛠️ Technology Stack

- **Backend**: Go + Fiber + MongoDB + JWT + bcrypt
- **Frontend**: Next.js + TypeScript + Tailwind + Ant Design

## 🔒 Security Features

- JWT token authentication
- bcrypt password hashing
- Input validation
- Protected routes
- Activity logging
- Role-based access control

## 📚 API Endpoints

- `POST /api/auth/login` - Login
- `GET /api/profile` - Get user profile (protected)
- `POST /api/change-password` - Change password (protected)
- `POST /api/logout` - Logout (protected)

Backend is currently running on port 3033 in demo mode. Connect MongoDB for full functionality.
