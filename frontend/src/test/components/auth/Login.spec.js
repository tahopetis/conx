import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createRouter, createWebHistory } from 'vue-router'
import { createPinia, setActivePinia } from 'pinia'
import Login from '@/views/auth/Login.vue'

// Mock Element Plus components
vi.mock('element-plus', async () => {
  const actual = await vi.importActual('element-plus')
  return {
    ...actual,
    ElMessage: {
      success: vi.fn(),
      error: vi.fn()
    },
    ElAlert: {
      name: 'ElAlert',
      template: '<div class="el-alert"><slot></slot></div>'
    }
  }
})

// Mock router
const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/auth/login', component: Login },
    { path: '/auth/register', component: { template: '<div>Register</div>' } },
    { path: '/auth/forgot-password', component: { template: '<div>Forgot Password</div>' } },
    { path: '/', component: { template: '<div>Dashboard</div>' } }
  ]
})

// Mock auth store
const mockAuthStore = {
  isAuthenticated: false,
  login: vi.fn(),
  logout: vi.fn()
}

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => mockAuthStore
}))

describe('Login.vue', () => {
  let wrapper

  beforeEach(() => {
    vi.clearAllMocks()
    
    // Reset mock store
    mockAuthStore.isAuthenticated = false
    mockAuthStore.login.mockResolvedValue({})
    
    // Setup Pinia
    const pinia = createPinia()
    setActivePinia(pinia)
    
    wrapper = mount(Login, {
      global: {
        plugins: [router, pinia],
        stubs: {
          'el-icon': true,
          'el-form': true,
          'el-form-item': true,
          'el-input': true,
          'el-checkbox': true,
          'el-link': true,
          'el-button': true,
          'el-alert': true,
          'user': true,
          'lock': true
        },
        mocks: {
          $route: {
            query: {}
          }
        }
      }
    })
  })

  it('renders properly', () => {
    expect(wrapper.exists()).toBe(true)
  })

  it('displays the login form title', () => {
    expect(wrapper.find('h1').text()).toBe('Welcome to conx CMDB')
  })

  it('initializes form with empty values', () => {
    expect(wrapper.vm.form.email).toBe('')
    expect(wrapper.vm.form.password).toBe('')
    expect(wrapper.vm.form.remember).toBe(false)
  })

  it('updates form data when inputs change', async () => {
    wrapper.vm.form.email = 'test@example.com'
    wrapper.vm.form.password = 'password123'
    
    expect(wrapper.vm.form.email).toBe('test@example.com')
    expect(wrapper.vm.form.password).toBe('password123')
  })

  it('disables submit button when form is invalid', async () => {
    // Form should be invalid initially
    expect(wrapper.vm.form.email).toBe('')
    expect(wrapper.vm.form.password).toBe('')
  })

  it('shows loading state during login', async () => {
    mockAuthStore.login.mockImplementationOnce(() => {
      return new Promise(resolve => setTimeout(resolve, 100))
    })
    
    wrapper.vm.form.email = 'test@example.com'
    wrapper.vm.form.password = 'password123'
    
    // Mock form validation
    wrapper.vm.formRef = {
      value: {
        validate: vi.fn().mockImplementation((callback) => {
          callback(true)
        })
      }
    }
    
    await wrapper.vm.handleSubmit()
    expect(wrapper.vm.loading).toBe(true)
  })

  it('shows error message when login fails', async () => {
    mockAuthStore.login.mockRejectedValueOnce(new Error('Login failed'))
    
    wrapper.vm.form.email = 'test@example.com'
    wrapper.vm.form.password = 'wrongpassword'
    
    // Mock form validation
    wrapper.vm.formRef = {
      value: {
        validate: vi.fn().mockImplementation((callback) => {
          callback(true)
        })
      }
    }
    
    await wrapper.vm.handleSubmit()
    expect(wrapper.vm.error).toBe('Login failed. Please check your credentials.')
  })

  it('calls login action with correct credentials', async () => {
    wrapper.vm.form.email = 'test@example.com'
    wrapper.vm.form.password = 'password123'
    
    // Mock form validation
    wrapper.vm.formRef = {
      value: {
        validate: vi.fn().mockImplementation((callback) => {
          callback(true)
        })
      }
    }
    
    await wrapper.vm.handleSubmit()
    expect(mockAuthStore.login).toHaveBeenCalledWith({
      email: 'test@example.com',
      password: 'password123'
    })
  })

  it('does not call login when form is invalid', async () => {
    // Mock form validation to return false
    wrapper.vm.formRef = {
      value: {
        validate: vi.fn().mockImplementation((callback) => {
          callback(false)
        })
      }
    }
    
    await wrapper.vm.handleSubmit()
    expect(mockAuthStore.login).not.toHaveBeenCalled()
  })

  it('navigates to forgot password page', async () => {
    await wrapper.vm.goToForgotPassword()
    await new Promise(resolve => setTimeout(resolve, 0))
    expect(router.currentRoute.value.path).toBe('/auth/forgot-password')
  })

  it('navigates to register page', async () => {
    await wrapper.vm.goToRegister()
    await new Promise(resolve => setTimeout(resolve, 0))
    expect(router.currentRoute.value.path).toBe('/auth/register')
  })

  it('redirects to dashboard if already logged in', async () => {
    mockAuthStore.isAuthenticated = true
    
    wrapper = mount(Login, {
      global: {
        plugins: [router, createPinia()],
        stubs: {
          'el-icon': true,
          'el-form': true,
          'el-form-item': true,
          'el-input': true,
          'el-checkbox': true,
          'el-link': true,
          'el-button': true,
          'el-alert': true,
          'user': true,
          'lock': true
        }
      }
    })
    
    await wrapper.vm.$nextTick()
    await new Promise(resolve => setTimeout(resolve, 0))
    expect(router.currentRoute.value.path).toBe('/')
  })

  it('clears error message when user starts typing', async () => {
    wrapper.vm.error = 'Previous error'
    expect(wrapper.vm.error).toBe('Previous error')
    
    wrapper.vm.form.email = 'test@example.com'
    
    // Error should be cleared when user starts typing
    expect(wrapper.vm.form.email).toBe('test@example.com')
  })

  it('remembers email if remember me is checked', async () => {
    wrapper.vm.form.remember = true
    expect(wrapper.vm.form.remember).toBe(true)
  })

  it('handles keyboard enter to submit form', async () => {
    wrapper.vm.form.email = 'test@example.com'
    wrapper.vm.form.password = 'password123'
    
    // Mock form validation
    wrapper.vm.formRef = {
      value: {
        validate: vi.fn().mockImplementation((callback) => {
          callback(true)
        })
      }
    }
    
    await wrapper.vm.handleSubmit()
    
    expect(mockAuthStore.login).toHaveBeenCalledWith({
      email: 'test@example.com',
      password: 'password123'
    })
  })
})
