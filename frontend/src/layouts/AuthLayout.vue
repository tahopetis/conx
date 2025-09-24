<template>
  <div class="auth-container">
    <div class="auth-left">
      <div class="auth-content">
        <div class="auth-logo">
          <div class="logo-icon">C</div>
          <h1>conx CMDB</h1>
        </div>
        
        <div class="auth-description">
          <h2>Configuration Management Database</h2>
          <p>Manage your IT infrastructure with ease and efficiency. Track configuration items, visualize relationships, and maintain complete control over your technology ecosystem.</p>
        </div>
        
        <div class="auth-features">
          <div class="feature-item">
            <el-icon><Monitor /></el-icon>
            <div>
              <h4>Comprehensive CI Management</h4>
              <p>Track and manage all configuration items in one place</p>
            </div>
          </div>
          
          <div class="feature-item">
            <el-icon><Connection /></el-icon>
            <div>
              <h4>Graph Visualization</h4>
              <p>Visualize complex relationships between infrastructure components</p>
            </div>
          </div>
          
          <div class="feature-item">
            <el-icon><Shield /></el-icon>
            <div>
              <h4>Enterprise Security</h4>
              <p>Role-based access control and secure authentication</p>
            </div>
          </div>
        </div>
      </div>
    </div>
    
    <div class="auth-right">
      <div class="auth-form-container">
        <div class="auth-form-header">
          <h2>{{ formTitle }}</h2>
          <p>{{ formSubtitle }}</p>
        </div>
        
        <div class="auth-form-content">
          <slot />
        </div>
        
        <div class="auth-form-footer">
          <div class="auth-links">
            <template v-if="$route.name === 'Login'">
              <p>Don't have an account? <router-link to="/auth/register">Sign up</router-link></p>
              <p><router-link to="/auth/forgot-password">Forgot your password?</router-link></p>
            </template>
            
            <template v-else-if="$route.name === 'Register'">
              <p>Already have an account? <router-link to="/auth/login">Sign in</router-link></p>
            </template>
            
            <template v-else-if="$route.name === 'ForgotPassword'">
              <p>Remember your password? <router-link to="/auth/login">Sign in</router-link></p>
            </template>
            
            <template v-else-if="$route.name === 'ResetPassword'">
              <p>Remember your password? <router-link to="/auth/login">Sign in</router-link></p>
            </template>
          </div>
          
          <div class="auth-version">
            <p>conx CMDB v1.0.0</p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { Monitor, Connection, Shield } from '@element-plus/icons-vue'

const route = useRoute()

const formTitle = computed(() => {
  switch (route.name) {
    case 'Login':
      return 'Welcome Back'
    case 'Register':
      return 'Create Account'
    case 'ForgotPassword':
      return 'Reset Password'
    case 'ResetPassword':
      return 'Set New Password'
    default:
      return 'Authentication'
  }
})

const formSubtitle = computed(() => {
  switch (route.name) {
    case 'Login':
      return 'Sign in to your account to continue'
    case 'Register':
      return 'Create a new account to get started'
    case 'ForgotPassword':
      return 'Enter your email to receive password reset instructions'
    case 'ResetPassword':
      return 'Create a new password for your account'
    default:
      return 'Please authenticate to continue'
  }
})
</script>

<style scoped>
.auth-container {
  display: flex;
  min-height: 100vh;
  width: 100vw;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.auth-left {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 60px;
  background: rgba(255, 255, 255, 0.1);
  backdrop-filter: blur(10px);
}

.auth-right {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 60px;
  background: rgba(255, 255, 255, 0.95);
}

.auth-content {
  max-width: 500px;
  text-align: center;
  color: white;
}

.auth-logo {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 16px;
  margin-bottom: 40px;
}

.logo-icon {
  width: 48px;
  height: 48px;
  background-color: rgba(255, 255, 255, 0.2);
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-weight: bold;
  font-size: 24px;
}

.auth-logo h1 {
  font-size: 32px;
  font-weight: 700;
  margin: 0;
  background: linear-gradient(135deg, #ffffff 0%, #e0e7ff 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.auth-description h2 {
  font-size: 28px;
  font-weight: 600;
  margin-bottom: 16px;
  color: white;
}

.auth-description p {
  font-size: 16px;
  line-height: 1.6;
  color: rgba(255, 255, 255, 0.9);
  margin-bottom: 48px;
}

.auth-features {
  display: flex;
  flex-direction: column;
  gap: 24px;
  text-align: left;
}

.feature-item {
  display: flex;
  align-items: flex-start;
  gap: 16px;
}

.feature-item .el-icon {
  font-size: 24px;
  color: rgba(255, 255, 255, 0.9);
  margin-top: 2px;
}

.feature-item h4 {
  font-size: 16px;
  font-weight: 600;
  margin: 0 0 4px 0;
  color: white;
}

.feature-item p {
  font-size: 14px;
  color: rgba(255, 255, 255, 0.8);
  margin: 0;
  line-height: 1.4;
}

.auth-form-container {
  width: 100%;
  max-width: 400px;
}

.auth-form-header {
  text-align: center;
  margin-bottom: 32px;
}

.auth-form-header h2 {
  font-size: 28px;
  font-weight: 600;
  color: #2c3e50;
  margin: 0 0 8px 0;
}

.auth-form-header p {
  font-size: 16px;
  color: #606266;
  margin: 0;
}

.auth-form-content {
  margin-bottom: 32px;
}

.auth-form-footer {
  text-align: center;
}

.auth-links {
  margin-bottom: 24px;
}

.auth-links p {
  font-size: 14px;
  color: #606266;
  margin: 8px 0;
}

.auth-links a {
  color: #409eff;
  text-decoration: none;
  font-weight: 500;
}

.auth-links a:hover {
  text-decoration: underline;
}

.auth-version p {
  font-size: 12px;
  color: #909399;
  margin: 0;
}

/* Responsive design */
@media (max-width: 1024px) {
  .auth-container {
    flex-direction: column;
  }
  
  .auth-left,
  .auth-right {
    flex: none;
    min-height: 50vh;
  }
  
  .auth-left {
    padding: 40px 20px;
  }
  
  .auth-right {
    padding: 40px 20px;
  }
  
  .auth-description {
    margin-bottom: 32px;
  }
  
  .auth-features {
    gap: 16px;
  }
}

@media (max-width: 768px) {
  .auth-logo h1 {
    font-size: 24px;
  }
  
  .auth-description h2 {
    font-size: 22px;
  }
  
  .auth-features {
    display: none;
  }
  
  .auth-form-container {
    max-width: 100%;
  }
}

@media (max-width: 480px) {
  .auth-left,
  .auth-right {
    padding: 20px;
  }
  
  .auth-logo {
    margin-bottom: 24px;
  }
  
  .auth-description {
    margin-bottom: 24px;
  }
  
  .auth-description h2 {
    font-size: 20px;
  }
  
  .auth-description p {
    font-size: 14px;
  }
}
</style>
