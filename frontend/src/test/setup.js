import { vi } from 'vitest'
import { config } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import * as ElementPlusIconsVue from '@element-plus/icons-vue'

// Mock window.matchMedia
Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: vi.fn().mockImplementation(query => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: vi.fn(), // deprecated
    removeListener: vi.fn(), // deprecated
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  })),
})

// Mock ResizeObserver
global.ResizeObserver = vi.fn().mockImplementation(() => ({
  observe: vi.fn(),
  unobserve: vi.fn(),
  disconnect: vi.fn(),
}))

// Mock IntersectionObserver
global.IntersectionObserver = vi.fn().mockImplementation(() => ({
  observe: vi.fn(),
  unobserve: vi.fn(),
  disconnect: vi.fn(),
}))

// Setup Vue Test Utils
config.global.plugins = [ElementPlus]

// Register Element Plus icons globally
for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
  config.global.components[key] = component
}

// Mock console methods to reduce noise in tests
global.console = {
  ...console,
  // Uncomment to ignore a specific log level
  // log: vi.fn(),
  // warn: vi.fn(),
  // error: vi.fn(),
}

// Mock axios with proper interceptors
const mockAxios = {
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
  delete: vi.fn(),
  create: vi.fn(() => mockAxios),
  interceptors: {
    request: {
      use: vi.fn(),
      eject: vi.fn()
    },
    response: {
      use: vi.fn(),
      eject: vi.fn()
    }
  }
}

vi.mock('axios', () => ({
  default: mockAxios
}))

// Mock vue-router
vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: vi.fn(),
    replace: vi.fn(),
    go: vi.fn(),
    back: vi.fn(),
    forward: vi.fn(),
  }),
  useRoute: () => ({
    params: {},
    query: {},
    path: '/',
    name: 'home',
  }),
  RouterLink: {
    name: 'RouterLink',
    template: '<a><slot /></a>',
  },
  createRouter: vi.fn(() => ({
    push: vi.fn(),
    replace: vi.fn(),
    go: vi.fn(),
    back: vi.fn(),
    forward: vi.fn(),
  })),
  createWebHistory: vi.fn(),
}))

// Mock pinia with proper exports
const mockCreatePinia = vi.fn(() => ({
  install: vi.fn(),
  state: {},
}))

const mockSetActivePinia = vi.fn()

const mockDefineStore = vi.fn((name, options) => {
  return vi.fn(() => ({
    ...options,
    $state: options.state ? options.state() : {},
    $patch: vi.fn(),
    $reset: vi.fn(),
    $onAction: vi.fn(),
  }))
})

vi.mock('pinia', () => ({
  defineStore: mockDefineStore,
  createPinia: mockCreatePinia,
  setActivePinia: mockSetActivePinia,
}))

// Mock dayjs with proper default export
const mockDayjsFn = vi.fn().mockImplementation((date = null) => {
  const defaultDate = date ? new Date(date) : new Date()
  
  return {
    format: vi.fn((formatStr = 'YYYY-MM-DD HH:mm:ss') => {
      if (formatStr === 'YYYY-MM-DD') {
        if (date === '2023-01-15T10:30:00Z') return '2023-01-15'
        return defaultDate.toISOString().split('T')[0]
      }
      if (date === '2023-01-15T10:30:00Z') return '2023-01-15 10:30:00'
      if (date === '2023-01-01T00:00:00Z') return '2023-01-01 00:00:00'
      return defaultDate.toISOString().replace('T', ' ').split('.')[0]
    }),
    fromNow: vi.fn(() => '2 days ago'),
    add: vi.fn(() => mockDayjsFn()),
    subtract: vi.fn(() => mockDayjsFn()),
    diff: vi.fn(() => 3600),
  }
})

mockDayjsFn.extend = vi.fn()

vi.mock('dayjs', () => ({
  default: mockDayjsFn,
  extend: vi.fn()
}))

// Mock nprogress
vi.mock('nprogress', () => ({
  default: {
    start: vi.fn(),
    done: vi.fn(),
    set: vi.fn(),
    inc: vi.fn(),
    configure: vi.fn(),
  },
}))

// Mock lodash-es
vi.mock('lodash-es', () => ({
  debounce: vi.fn((fn) => fn),
  throttle: vi.fn((fn) => fn),
  cloneDeep: vi.fn((obj) => JSON.parse(JSON.stringify(obj))),
  get: vi.fn((obj, path) => path.split('.').reduce((o, p) => o?.[p], obj)),
  set: vi.fn((obj, path, value) => {
    path.split('.').reduce((o, p, i, arr) => {
      if (i === arr.length - 1) {
        o[p] = value
      } else {
        o[p] = o[p] || {}
      }
      return o[p]
    }, obj)
    return obj
  }),
}))
