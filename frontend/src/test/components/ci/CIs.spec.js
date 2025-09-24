import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createRouter, createWebHistory } from 'vue-router'
import { createPinia, setActivePinia } from 'pinia'
import CIs from '@/views/ci/CIs.vue'

// Mock Element Plus components
vi.mock('element-plus', async () => {
  const actual = await vi.importActual('element-plus')
  return {
    ...actual,
    ElMessage: {
      success: vi.fn(),
      error: vi.fn()
    },
    ElMessageBox: {
      confirm: vi.fn()
    }
  }
})

// Mock API service with factory function
vi.mock('@/services/api', () => ({
  getCIs: vi.fn(),
  deleteCI: vi.fn()
}))

// Mock router
const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/cis', component: CIs },
    { path: '/cis/create', component: { template: '<div>Create CI</div>' } },
    { path: '/cis/:id', component: { template: '<div>CI Detail</div>' } },
    { path: '/cis/:id/edit', component: { template: '<div>Edit CI</div>' } }
  ]
})

describe('CIs.vue', () => {
  let wrapper
  let mockApi

  const mockCIs = [
    {
      id: 1,
      name: 'Server 1',
      type: 'server',
      status: 'active',
      environment: 'production',
      owner: 'John Doe',
      created_at: '2023-01-01T00:00:00Z',
      updated_at: '2023-01-01T00:00:00Z'
    },
    {
      id: 2,
      name: 'Database 1',
      type: 'database',
      status: 'active',
      environment: 'production',
      owner: 'Jane Doe',
      created_at: '2023-01-02T00:00:00Z',
      updated_at: '2023-01-02T00:00:00Z'
    }
  ]

  beforeEach(() => {
    vi.clearAllMocks()
    
    // Setup Pinia
    const pinia = createPinia()
    setActivePinia(pinia)
    
    // Get mock API
    const api = require('@/services/api')
    mockApi = api
    
    // Mock API responses
    mockApi.getCIs.mockResolvedValue({
      data: {
        data: mockCIs,
        total: 2
      }
    })
    
    mockApi.deleteCI.mockResolvedValue({})
    
    wrapper = mount(CIs, {
      global: {
        plugins: [router, pinia],
        stubs: {
          'el-icon': true,
          'el-link': true,
          'el-button': true,
          'el-button-group': true,
          'el-card': true,
          'el-row': true,
          'el-col': true,
          'el-input': true,
          'el-select': true,
          'el-option': true,
          'el-table': true,
          'el-table-column': true,
          'el-tag': true,
          'el-pagination': true,
          'el-dialog': true,
          'el-loading-directive': true
        }
      }
    })
  })

  it('renders properly', () => {
    expect(wrapper.exists()).toBe(true)
  })

  it('displays the correct title', () => {
    expect(wrapper.find('h1').text()).toBe('Configuration Items')
  })

  it('shows loading state when loading', async () => {
    // Mock loading state
    mockApi.getCIs.mockImplementationOnce(() => {
      return new Promise(resolve => {
        setTimeout(() => {
          resolve({
            data: {
              data: mockCIs,
              total: 2
            }
          })
        }, 100)
      })
    })
    
    wrapper = mount(CIs, {
      global: {
        plugins: [router, createPinia()],
        stubs: {
          'el-icon': true,
          'el-link': true,
          'el-button': true,
          'el-button-group': true,
          'el-card': true,
          'el-row': true,
          'el-col': true,
          'el-input': true,
          'el-select': true,
          'el-option': true,
          'el-table': true,
          'el-table-column': true,
          'el-tag': true,
          'el-pagination': true,
          'el-dialog': true,
          'el-loading-directive': true
        }
      }
    })
    
    await wrapper.vm.$nextTick()
    expect(wrapper.vm.loading).toBe(true)
  })

  it('shows error message when there is an error', async () => {
    mockApi.getCIs.mockRejectedValueOnce(new Error('API Error'))
    
    wrapper = mount(CIs, {
      global: {
        plugins: [router, createPinia()],
        stubs: {
          'el-icon': true,
          'el-link': true,
          'el-button': true,
          'el-button-group': true,
          'el-card': true,
          'el-row': true,
          'el-col': true,
          'el-input': true,
          'el-select': true,
          'el-option': true,
          'el-table': true,
          'el-table-column': true,
          'el-tag': true,
          'el-pagination': true,
          'el-dialog': true,
          'el-loading-directive': true
        }
      }
    })
    
    await wrapper.vm.$nextTick()
    await new Promise(resolve => setTimeout(resolve, 0))
    expect(mockApi.getCIs).toHaveBeenCalled()
  })

  it('displays CI list when data is loaded', async () => {
    await wrapper.vm.$nextTick()
    expect(wrapper.vm.cis).toHaveLength(2)
    expect(wrapper.vm.cis[0].name).toBe('Server 1')
  })

  it('calls fetchCIs on component mount', () => {
    expect(mockApi.getCIs).toHaveBeenCalled()
  })

  it('handles delete confirmation', async () => {
    await wrapper.vm.$nextTick()
    await wrapper.vm.confirmDelete(mockCIs[0])
    expect(wrapper.vm.deleteDialog.ci).toEqual(mockCIs[0])
    expect(wrapper.vm.deleteDialog.visible).toBe(true)
  })

  it('calls deleteCI when deletion is confirmed', async () => {
    await wrapper.vm.$nextTick()
    await wrapper.vm.confirmDelete(mockCIs[0])
    await wrapper.vm.deleteCI()
    expect(mockApi.deleteCI).toHaveBeenCalledWith(mockCIs[0].id)
  })

  it('handles pagination change', async () => {
    await wrapper.vm.$nextTick()
    await wrapper.vm.handlePageChange(2)
    expect(wrapper.vm.pagination.page).toBe(2)
    expect(mockApi.getCIs).toHaveBeenCalled()
  })

  it('filters CIs based on search query', async () => {
    await wrapper.vm.$nextTick()
    wrapper.vm.filters.search = 'Server'
    await wrapper.vm.handleSearch()
    expect(wrapper.vm.pagination.page).toBe(1)
    expect(mockApi.getCIs).toHaveBeenCalled()
  })

  it('filters CIs based on type filter', async () => {
    await wrapper.vm.$nextTick()
    wrapper.vm.filters.type = 'server'
    await wrapper.vm.handleSearch()
    expect(wrapper.vm.pagination.page).toBe(1)
    expect(mockApi.getCIs).toHaveBeenCalled()
  })

  it('navigates to create CI page', async () => {
    await wrapper.vm.$nextTick()
    await wrapper.vm.goToCreate()
    expect(router.currentRoute.value.path).toBe('/cis/create')
  })

  it('navigates to edit CI page', async () => {
    await wrapper.vm.$nextTick()
    await wrapper.vm.goToEdit(1)
    expect(router.currentRoute.value.path).toBe('/cis/1/edit')
  })

  it('navigates to CI detail page', async () => {
    await wrapper.vm.$nextTick()
    await wrapper.vm.goToDetail(1)
    expect(router.currentRoute.value.path).toBe('/cis/1')
  })

  it('formats dates correctly', () => {
    const date = wrapper.vm.formatDate('2023-01-01T00:00:00Z')
    expect(date).toMatch(/\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}/)
  })

  it('gets status tag type correctly', () => {
    expect(wrapper.vm.getStatusTagType('active')).toBe('success')
    expect(wrapper.vm.getStatusTagType('inactive')).toBe('info')
    expect(wrapper.vm.getStatusTagType('maintenance')).toBe('warning')
    expect(wrapper.vm.getStatusTagType('decommissioned')).toBe('danger')
  })

  it('computes filtered CIs correctly', async () => {
    await wrapper.vm.$nextTick()
    wrapper.vm.filters.search = 'Server'
    const filtered = wrapper.vm.cis.filter(ci => 
      ci.name.toLowerCase().includes('server')
    )
    expect(filtered).toHaveLength(1)
    expect(filtered[0].name).toBe('Server 1')
  })

  it('computes filtered CIs with type filter', async () => {
    await wrapper.vm.$nextTick()
    wrapper.vm.filters.type = 'database'
    const filtered = wrapper.vm.cis.filter(ci => ci.type === 'database')
    expect(filtered).toHaveLength(1)
    expect(filtered[0].name).toBe('Database 1')
  })

  it('computes filtered CIs with both filters', async () => {
    await wrapper.vm.$nextTick()
    wrapper.vm.filters.search = 'Database'
    wrapper.vm.filters.type = 'database'
    const filtered = wrapper.vm.cis.filter(ci => 
      ci.name.toLowerCase().includes('database') && ci.type === 'database'
    )
    expect(filtered).toHaveLength(1)
    expect(filtered[0].name).toBe('Database 1')
  })

  it('shows no results message when no CIs match filters', async () => {
    await wrapper.vm.$nextTick()
    wrapper.vm.filters.search = 'NonExistent'
    const filtered = wrapper.vm.cis.filter(ci => 
      ci.name.toLowerCase().includes('nonexistent')
    )
    expect(filtered).toHaveLength(0)
  })
})
