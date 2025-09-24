import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

// Layouts
import MainLayout from '@/layouts/MainLayout.vue'
import AuthLayout from '@/layouts/AuthLayout.vue'

// Views
import Login from '@/views/auth/Login.vue'
import Register from '@/views/auth/Register.vue'
import ForgotPassword from '@/views/auth/ForgotPassword.vue'
import ResetPassword from '@/views/auth/ResetPassword.vue'

import Dashboard from '@/views/Dashboard.vue'
import CIs from '@/views/ci/CIs.vue'
import CIDetail from '@/views/ci/CIDetail.vue'
import CICreate from '@/views/ci/CICreate.vue'
import CIEdit from '@/views/ci/CIEdit.vue'
import Graph from '@/views/graph/Graph.vue'
import Search from '@/views/search/Search.vue'
import Profile from '@/views/user/Profile.vue'
import Settings from '@/views/user/Settings.vue'
import NotFound from '@/views/NotFound.vue'

const routes = [
  {
    path: '/',
    component: MainLayout,
    meta: { requiresAuth: true },
    children: [
      {
        path: '',
        name: 'Dashboard',
        component: Dashboard,
        meta: { title: 'Dashboard' }
      },
      {
        path: 'cis',
        name: 'CIs',
        component: CIs,
        meta: { title: 'Configuration Items' }
      },
      {
        path: 'cis/create',
        name: 'CICreate',
        component: CICreate,
        meta: { title: 'Create Configuration Item' }
      },
      {
        path: 'cis/:id',
        name: 'CIDetail',
        component: CIDetail,
        meta: { title: 'Configuration Item Details' }
      },
      {
        path: 'cis/:id/edit',
        name: 'CIEdit',
        component: CIEdit,
        meta: { title: 'Edit Configuration Item' }
      },
      {
        path: 'graph',
        name: 'Graph',
        component: Graph,
        meta: { title: 'Graph Visualization' }
      },
      {
        path: 'search',
        name: 'Search',
        component: Search,
        meta: { title: 'Search' }
      },
      {
        path: 'profile',
        name: 'Profile',
        component: Profile,
        meta: { title: 'Profile' }
      },
      {
        path: 'settings',
        name: 'Settings',
        component: Settings,
        meta: { title: 'Settings' }
      }
    ]
  },
  {
    path: '/auth',
    component: AuthLayout,
    meta: { guest: true },
    children: [
      {
        path: 'login',
        name: 'Login',
        component: Login,
        meta: { title: 'Login' }
      },
      {
        path: 'register',
        name: 'Register',
        component: Register,
        meta: { title: 'Register' }
      },
      {
        path: 'forgot-password',
        name: 'ForgotPassword',
        component: ForgotPassword,
        meta: { title: 'Forgot Password' }
      },
      {
        path: 'reset-password/:token',
        name: 'ResetPassword',
        component: ResetPassword,
        meta: { title: 'Reset Password' }
      }
    ]
  },
  {
    path: '/:pathMatch(.*)*',
    name: 'NotFound',
    component: NotFound
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

// Navigation guard
router.beforeEach((to, from, next) => {
  const authStore = useAuthStore()
  
  // Update document title
  if (to.meta.title) {
    document.title = `${to.meta.title} - conx CMDB`
  } else {
    document.title = 'conx CMDB'
  }
  
  // Check if route requires authentication
  if (to.meta.requiresAuth && !authStore.isAuthenticated) {
    next('/auth/login')
    return
  }
  
  // Check if route is for guests only (like login page)
  if (to.meta.guest && authStore.isAuthenticated) {
    next('/')
    return
  }
  
  // Check role-based access
  if (to.meta.roles && !authStore.hasAnyRole(to.meta.roles)) {
    next('/?error=unauthorized')
    return
  }
  
  next()
})

export default router
