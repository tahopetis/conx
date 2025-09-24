# Authentication System Documentation

## Overview

The conx CMDB API implements a comprehensive authentication and authorization system using JWT (JSON Web Tokens) and role-based access control (RBAC). This document provides detailed information about the authentication architecture, endpoints, and usage.

## Architecture

### Components

1. **JWT Service** (`internal/auth/jwt.go`)
   - Handles token generation and validation
   - Supports access tokens and refresh tokens
   - Implements secure token signing and verification

2. **Password Service** (`internal/auth/password.go`)
   - Implements Argon2id password hashing
   - Provides password strength validation
   - Includes password generation and pattern detection

3. **Authentication Middleware** (`internal/auth/middleware.go`)
   - Handles request authentication
   - Supports role-based and permission-based authorization
   - Provides context injection for user information

4. **User Repository** (`internal/repositories/user_repository.go`)
   - Manages user data persistence
   - Handles user CRUD operations
   - Implements authentication logic

5. **Authentication Handlers** (`internal/api/auth_handlers.go`)
   - Exposes authentication endpoints
   - Handles registration, login, and token refresh
   - Manages password reset flows

### Security Features

- **JWT Tokens**: Stateless authentication with configurable TTL
- **Argon2id Hashing**: Memory-hard password hashing algorithm
- **Password Strength**: Comprehensive password validation
- **Role-Based Access**: Granular permission control
- **Token Refresh**: Secure token renewal mechanism
- **Rate Limiting**: Protection against brute force attacks
- **CORS Support**: Configurable cross-origin resource sharing

## Authentication Flow

### 1. User Registration

```http
POST /api/v1/auth/register
Content-Type: application/json

{
  "username": "newuser",
  "email": "user@example.com",
  "password": "SecurePassword123!",
  "first_name": "John",
  "last_name": "Doe"
}
```

**Response:**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "Bearer",
  "expires_in": 900,
  "user": {
    "id": "uuid-here",
    "username": "newuser",
    "email": "user@example.com",
    "first_name": "John",
    "last_name": "Doe",
    "is_active": true,
    "is_verified": false,
    "roles": ["viewer"]
  }
}
```

### 2. User Login

```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "newuser",
  "password": "SecurePassword123!"
}
```

**Response:**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "Bearer",
  "expires_in": 900,
  "user": {
    "id": "uuid-here",
    "username": "newuser",
    "email": "user@example.com",
    "first_name": "John",
    "last_name": "Doe",
    "is_active": true,
    "is_verified": false,
    "roles": ["viewer", "editor"]
  }
}
```

### 3. Token Refresh

```http
POST /api/v1/auth/refresh
Content-Type: application/json

{
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Response:**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "Bearer",
  "expires_in": 900,
  "user": {
    "id": "uuid-here",
    "username": "newuser",
    "email": "user@example.com",
    "first_name": "John",
    "last_name": "Doe",
    "is_active": true,
    "is_verified": false,
    "roles": ["viewer", "editor"]
  }
}
```

### 4. Password Reset Request

```http
POST /api/v1/auth/password-reset-request
Content-Type: application/json

{
  "email": "user@example.com"
}
```

**Response:**
```json
{
  "message": "If your email is registered, you will receive a password reset link"
}
```

### 5. Password Reset Confirmation

```http
POST /api/v1/auth/password-reset
Content-Type: application/json

{
  "token": "reset-token-here",
  "new_password": "NewSecurePassword123!"
}
```

**Response:**
```json
{
  "message": "Password reset successfully"
}
```

## Protected Endpoints

All protected endpoints require a valid JWT token in the Authorization header:

```
Authorization: Bearer <access_token>
```

### User Profile Management

#### Get Profile
```http
GET /api/v1/auth/profile
Authorization: Bearer <access_token>
```

#### Update Profile
```http
PUT /api/v1/auth/profile
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "first_name": "Updated",
  "last_name": "Name",
  "email": "updated@example.com"
}
```

#### Change Password
```http
POST /api/v1/auth/change-password
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "current_password": "CurrentPassword123!",
  "new_password": "NewSecurePassword123!"
}
```

#### Logout
```http
POST /api/v1/auth/logout
Authorization: Bearer <access_token>
```

## Roles and Permissions

### Default Roles

The system includes the following default roles:

1. **admin** - Full system access
   - All CI operations (create, read, update, delete)
   - User and role management
   - System configuration access
   - Audit log access

2. **ci_manager** - Configuration item management
   - All CI operations (create, read, update, delete)
   - Relationship management
   - Import/Export capabilities

3. **viewer** - Read-only access
   - CI read access
   - Basic viewing capabilities

4. **auditor** - Audit and monitoring
   - CI read access
   - Audit log access

### Permission Mapping

| Role | CI CRUD | Relationships | Users | Roles | Permissions | Audit | Import/Export |
|------|---------|--------------|-------|-------|-------------|-------|---------------|
| admin | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| ci_manager | ✓ | ✓ | ✗ | ✗ | ✗ | ✗ | ✓ |
| viewer | R | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ |
| auditor | R | ✗ | ✗ | ✗ | ✗ | ✓ | ✗ |

### Using Authorization Middleware

The authentication system provides several middleware options:

#### 1. Basic Authentication
```go
r.Use(auth.NewAuthMiddleware(auth.AuthConfig{
    JWTService: jwtService,
    Logger:     logger,
    ExcludePaths: []string{"/public"},
}).Middleware)
```

#### 2. Role-Based Authorization
```go
r.Use(auth.RequireRole("admin"))
```

#### 3. Permission-Based Authorization
```go
r.Use(auth.RequirePermission("ci:create"))
```

#### 4. Optional Authentication
```go
r.Use(auth.OptionalAuthMiddleware(jwtService, logger))
```

## Configuration

### Authentication Configuration

```yaml
auth:
  secret_key: "your-secret-key-change-in-production"
  access_token_ttl: "15m"
  refresh_token_ttl: "7d"
  password_min_length: 8
  password_max_length: 128
  max_login_attempts: 5
  lockout_duration: "15m"
```

### Password Requirements

- Minimum length: 8 characters
- Maximum length: 128 characters
- Must contain at least one uppercase letter
- Must contain at least one lowercase letter
- Must contain at least one number
- Must contain at least one special character
- Cannot contain common patterns or sequential characters

### Security Best Practices

1. **Production Secret Key**
   - Use a cryptographically secure random key
   - Minimum 32 characters long
   - Store securely using environment variables or secret management

2. **Token TTL Configuration**
   - Access tokens: 15 minutes (recommended)
   - Refresh tokens: 7 days (recommended)
   - Adjust based on security requirements

3. **Password Policy**
   - Enforce strong password requirements
   - Implement password expiration
   - Use secure password hashing (Argon2id)

4. **Rate Limiting**
   - Implement login attempt limits
   - Use progressive delays for failed attempts
   - Consider IP-based blocking

## Error Handling

### Authentication Errors

| Error Code | HTTP Status | Description |
|------------|-------------|-------------|
| `invalid_credentials` | 401 | Invalid username or password |
| `account_inactive` | 403 | User account is deactivated |
| `account_locked` | 423 | Account temporarily locked |
| `token_expired` | 401 | JWT token has expired |
| `invalid_token` | 401 | Invalid JWT token |
| `user_not_found` | 404 | User does not exist |

### Validation Errors

| Error Code | HTTP Status | Description |
|------------|-------------|-------------|
| `invalid_request` | 400 | Malformed request body |
| `invalid_email` | 400 | Invalid email format |
| `weak_password` | 400 | Password does not meet requirements |
| `password_mismatch` | 400 | Current password does not match |

## Testing

### Running Authentication Tests

```bash
# Run all authentication tests
go test ./internal/auth/...

# Run with verbose output
go test -v ./internal/auth/...

# Run specific test
go test -run TestJWTService ./internal/auth/
```

### Test Coverage

The authentication system includes comprehensive tests covering:

- JWT token generation and validation
- Password hashing and verification
- Password strength validation
- Authentication middleware
- User model validation
- Error handling scenarios

## Integration Examples

### Frontend Integration (JavaScript)

```javascript
// User registration
async function register(userData) {
  const response = await fetch('/api/v1/auth/register', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(userData),
  });
  
  if (!response.ok) {
    throw new Error('Registration failed');
  }
  
  return await response.json();
}

// User login
async function login(credentials) {
  const response = await fetch('/api/v1/auth/login', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(credentials),
  });
  
  if (!response.ok) {
    throw new Error('Login failed');
  }
  
  const data = await response.json();
  
  // Store tokens
  localStorage.setItem('accessToken', data.access_token);
  localStorage.setItem('refreshToken', data.refresh_token);
  
  return data;
}

// Protected API call
async function fetchProtectedData() {
  const accessToken = localStorage.getItem('accessToken');
  
  const response = await fetch('/api/v1/cis', {
    headers: {
      'Authorization': `Bearer ${accessToken}`,
    },
  });
  
  if (!response.ok) {
    // Handle token refresh
    if (response.status === 401) {
      await refreshToken();
      return fetchProtectedData();
    }
    throw new Error('API request failed');
  }
  
  return await response.json();
}

// Token refresh
async function refreshToken() {
  const refreshToken = localStorage.getItem('refreshToken');
  
  const response = await fetch('/api/v1/auth/refresh', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ refresh_token: refreshToken }),
  });
  
  if (!response.ok) {
    // Redirect to login
    window.location.href = '/login';
    throw new Error('Token refresh failed');
  }
  
  const data = await response.json();
  localStorage.setItem('accessToken', data.access_token);
  
  return data;
}
```

### cURL Examples

```bash
# Register user
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "email": "test@example.com",
    "password": "SecurePassword123!",
    "first_name": "Test",
    "last_name": "User"
  }'

# Login
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "password": "SecurePassword123!"
  }'

# Access protected endpoint
curl -X GET http://localhost:8080/api/v1/auth/profile \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"

# Change password
curl -X POST http://localhost:8080/api/v1/auth/change-password \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "current_password": "SecurePassword123!",
    "new_password": "NewSecurePassword123!"
  }'
```

## Troubleshooting

### Common Issues

1. **Token Validation Fails**
   - Check if the secret key is consistent across all services
   - Verify the token is not expired
   - Ensure the token is properly formatted

2. **Password Hashing Errors**
   - Verify Argon2id parameters are correctly configured
   - Check if the password meets strength requirements
   - Ensure consistent hashing parameters

3. **Authentication Middleware Issues**
   - Verify the JWT token is included in the Authorization header
   - Check if the token format is correct (Bearer <token>)
   - Ensure the middleware is properly configured

4. **Database Connection Issues**
   - Verify database connection parameters
   - Check if the user table exists and has the correct schema
   - Ensure proper database permissions

### Debug Mode

Enable debug logging to troubleshoot authentication issues:

```yaml
logging:
  level: "debug"
  format: "json"
  output: "stdout"
```

### Health Checks

Monitor authentication system health:

```bash
curl http://localhost:8080/api/v1/health
```

## Future Enhancements

Planned improvements to the authentication system:

1. **Multi-Factor Authentication (MFA)**
   - TOTP support
   - SMS verification
   - Email verification

2. **Social Login**
   - OAuth 2.0 integration
   - SAML support
   - Social provider integration

3. **Advanced Security Features**
   - Device fingerprinting
   - Location-based access control
   - Anomaly detection

4. **Session Management**
   - Real-time session monitoring
   - Session revocation
   - Concurrent session limits

5. **Compliance Features**
   - GDPR compliance tools
   - Audit trail enhancement
   - Data retention policies

## Contributing

When contributing to the authentication system:

1. Follow the established code patterns
2. Write comprehensive tests for new features
3. Update documentation for any changes
4. Ensure security best practices are followed
5. Test all authentication flows thoroughly

## Support

For issues or questions regarding the authentication system:

1. Check the troubleshooting section
2. Review the API documentation
3. Check existing issues
4. Create a new issue with detailed information
5. Contact the development team for critical issues
