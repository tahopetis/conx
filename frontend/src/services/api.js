import axios from 'axios'
import { useAuthStore } from '@/stores/auth'

// Create axios instance
const api = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Request interceptor
api.interceptors.request.use(
  (config) => {
    const authStore = useAuthStore()
    
    // Add auth token if available
    if (authStore.token) {
      config.headers.Authorization = `Bearer ${authStore.token}`
    }
    
    // Add loading state if needed
    // You can add a global loading indicator here
    
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// Response interceptor
api.interceptors.response.use(
  (response) => {
    // Any status code that lie within the range of 2xx cause this function to trigger
    return response
  },
  async (error) => {
    const authStore = useAuthStore()
    const originalRequest = error.config
    
    // Handle 401 Unauthorized errors
    if (error.response?.status === 401 && !originalRequest._retry) {
      originalRequest._retry = true
      
      try {
        // Try to refresh the token
        const newToken = await authStore.refreshTokens()
        
        // Update the request with new token
        originalRequest.headers.Authorization = `Bearer ${newToken}`
        
        // Retry the original request
        return api(originalRequest)
      } catch (refreshError) {
        // If refresh fails, logout and redirect to login
        authStore.logout()
        window.location.href = '/auth/login'
        return Promise.reject(refreshError)
      }
    }
    
    // Handle 403 Forbidden errors
    if (error.response?.status === 403) {
      // Redirect to unauthorized page or show error
      window.location.href = '/?error=unauthorized'
    }
    
    // Handle network errors
    if (!error.response) {
      console.error('Network Error:', error.message)
      // You can show a global network error message here
    }
    
    // Handle other errors
    const errorMessage = error.response?.data?.message || error.message || 'An error occurred'
    console.error('API Error:', errorMessage)
    
    return Promise.reject(error)
  }
)

// API service methods
export default {
  // Auth endpoints
  async login(credentials) {
    return api.post('/auth/login', credentials)
  },
  
  async register(userData) {
    return api.post('/auth/register', userData)
  },
  
  async logout() {
    return api.post('/auth/logout')
  },
  
  async getProfile() {
    return api.get('/auth/profile')
  },
  
  async updateProfile(profileData) {
    return api.put('/auth/profile', profileData)
  },
  
  async changePassword(passwordData) {
    return api.post('/auth/change-password', passwordData)
  },
  
  async forgotPassword(email) {
    return api.post('/auth/forgot-password', { email })
  },
  
  async resetPassword(token, passwordData) {
    return api.post(`/auth/reset-password/${token}`, passwordData)
  },
  
  async refreshToken(refreshToken) {
    return api.post('/auth/refresh', { refresh_token: refreshToken })
  },
  
  // User endpoints
  async getUsers(params = {}) {
    return api.get('/users', { params })
  },
  
  async getUser(id) {
    return api.get(`/users/${id}`)
  },
  
  async createUser(userData) {
    return api.post('/users', userData)
  },
  
  async updateUser(id, userData) {
    return api.put(`/users/${id}`, userData)
  },
  
  async deleteUser(id) {
    return api.delete(`/users/${id}`)
  },
  
  // Role endpoints
  async getRoles(params = {}) {
    return api.get('/roles', { params })
  },
  
  async getRole(id) {
    return api.get(`/roles/${id}`)
  },
  
  async createRole(roleData) {
    return api.post('/roles', roleData)
  },
  
  async updateRole(id, roleData) {
    return api.put(`/roles/${id}`, roleData)
  },
  
  async deleteRole(id) {
    return api.delete(`/roles/${id}`)
  },
  
  async assignRoleToUser(userId, roleId) {
    return api.post(`/users/${userId}/roles`, { role_id: roleId })
  },
  
  async revokeRoleFromUser(userId, roleId) {
    return api.delete(`/users/${userId}/roles/${roleId}`)
  },
  
  // Permission endpoints
  async getPermissions(params = {}) {
    return api.get('/permissions', { params })
  },
  
  async getPermission(id) {
    return api.get(`/permissions/${id}`)
  },
  
  async createPermission(permissionData) {
    return api.post('/permissions', permissionData)
  },
  
  async updatePermission(id, permissionData) {
    return api.put(`/permissions/${id}`, permissionData)
  },
  
  async deletePermission(id) {
    return api.delete(`/permissions/${id}`)
  },
  
  async grantPermissionToRole(roleId, permissionId) {
    return api.post(`/roles/${roleId}/permissions`, { permission_id: permissionId })
  },
  
  async revokePermissionFromRole(roleId, permissionId) {
    return api.delete(`/roles/${roleId}/permissions/${permissionId}`)
  },
  
  // Session endpoints
  async getSessions(params = {}) {
    return api.get('/sessions', { params })
  },
  
  async getSession(id) {
    return api.get(`/sessions/${id}`)
  },
  
  async revokeSession(id) {
    return api.delete(`/sessions/${id}`)
  },
  
  async revokeAllUserSessions(userId) {
    return api.delete(`/sessions/user/${userId}`)
  },
  
  async getSessionActivities(params = {}) {
    return api.get('/sessions/activities', { params })
  },
  
  async getSessionStats() {
    return api.get('/sessions/stats')
  },
  
  // CI endpoints
  async getCIs(params = {}) {
    return api.get('/cis', { params })
  },
  
  async getCI(id) {
    return api.get(`/cis/${id}`)
  },
  
  async createCI(ciData) {
    return api.post('/cis', ciData)
  },
  
  async updateCI(id, ciData) {
    return api.put(`/cis/${id}`, ciData)
  },
  
  async deleteCI(id) {
    return api.delete(`/cis/${id}`)
  },
  
  // Relationship endpoints
  async getRelationships(params = {}) {
    return api.get('/relationships', { params })
  },
  
  async createRelationship(relationshipData) {
    return api.post('/relationships', relationshipData)
  },
  
  async deleteRelationship(id) {
    return api.delete(`/relationships/${id}`)
  },
  
  // Graph endpoints
  async getGraphData(params = {}) {
    return api.get('/graph', { params })
  },
  
  async getSubgraph(nodeId, params = {}) {
    return api.get(`/graph/subgraph/${nodeId}`, { params })
  },
  
  async getGraphStats() {
    return api.get('/graph/stats')
  },
  
  // Search endpoints
  async search(query, params = {}) {
    return api.get('/search', { params: { q: query, ...params } })
  },
  
  // Generic HTTP methods
  get(url, config = {}) {
    return api.get(url, config)
  },
  
  post(url, data, config = {}) {
    return api.post(url, data, config)
  },
  
  put(url, data, config = {}) {
    return api.put(url, data, config)
  },
  
  delete(url, config = {}) {
    return api.delete(url, config)
  },
  
  patch(url, data, config = {}) {
    return api.patch(url, data, config)
  }
}
