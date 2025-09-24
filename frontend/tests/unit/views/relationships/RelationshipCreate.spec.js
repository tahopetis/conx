import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createRouter, createWebHistory, Router } from 'vue-router'
import { createTestingPinia } from '@pinia/testing'
import RelationshipCreate from '@/views/relationships/RelationshipCreate.vue'
import DynamicForm from '@/components/forms/DynamicForm.vue'

// Mock stores
const mockSchemaStore = {
  fetchRelationshipTypeSchemas: vi.fn().mockResolvedValue({
    schemas: [
      { id: 1, name: 'depends_on', attributes: [{ name: 'description', type: 'string', required: false }] },
      { id: 2, name: 'hosted_on', attributes: [{ name: 'port', type: 'number', required: true }] }
    ]
  }),
  fetchRelationshipTypeSchemaDetail: vi.fn().mockResolvedValue({
    id: 1,
    name: 'depends_on',
    attributes: [{ name: 'description', type: 'string', required: false }]
  })
}

const mockCIStore = {
  fetchCIs: vi.fn().mockResolvedValue({
    data: [
      { id: 1, name: 'Server 1', type: 'server' },
      { id: 2, name: 'Database 1', type: 'database' },
      { id: 3, name: 'Application 1', type: 'application' }
    ]
  })
}

const mockRelationshipStore = {
  createRelationship: vi.fn().mockResolvedValue({
    id: 1,
    source_id: 1,
    target_id: 2,
    schema_id: 1,
    schema_name: 'depends_on',
    description: 'Test relationship'
  })
}

describe('RelationshipCreate.vue', () => {
  let wrapper
  let pinia
  let router

  const createComponent = () => {
    return mount(RelationshipCreate, {
      global: {
        plugins: [pinia, router],
        mocks: {
          $route: { params: {} },
          $router: router
        },
        provide: {
          schemaStore: mockSchemaStore,
          ciStore: mockCIStore,
          relationshipStore: mockRelationshipStore
        },
        stubs: {
          DynamicForm: true,
          'v-card': true,
          'v-card-text': true,
          'v-row': true,
          'v-col': true,
          'v-select': true,
          'v-btn': true,
          'v-icon': true,
          'v-divider': true,
          'v-chip': true,
          'v-skeleton-loader': true
        }
      }
    })
  }

  beforeEach(() => {
    vi.clearAllMocks()
    
    pinia = createTestingPinia({
      createSpy: vi.fn,
      initialState: {
        schema: mockSchemaStore,
        ci: mockCIStore,
        relationship: mockRelationshipStore
      }
    })

    router = createRouter({
      history: createWebHistory(),
      routes: [
        { path: '/relationships', name: 'relationships' },
        { path: '/relationships/create', name: 'relationship-create' }
      ]
    })

    wrapper = createComponent()
  })

  afterEach(() => {
    wrapper.unmount()
  })

  it('renders properly', () => {
    expect(wrapper.exists()).toBe(true)
    expect(wrapper.find('h1').text()).toBe('Create Relationship')
  })

  it('loads available CIs on mount', async () => {
    expect(mockCIStore.fetchCIs).toHaveBeenCalledWith({
      page: 1,
      page_size: 1000,
      status: 'active'
    })
  })

  it('loads available schemas on mount', async () => {
    expect(mockSchemaStore.fetchRelationshipTypeSchemas).toHaveBeenCalledWith({
      page: 1,
      page_size: 100,
      is_active: true
    })
  })

  it('filters target CIs when source CI is selected', async () => {
    await wrapper.setData({
      availableCIs: [
        { id: 1, name: 'Server 1', type: 'server' },
        { id: 2, name: 'Database 1', type: 'database' },
        { id: 3, name: 'Application 1', type: 'application' }
      ]
    })

    await wrapper.vm.handleSourceCIChange(1)

    expect(wrapper.vm.availableTargetCIs).toEqual([
      { id: 2, name: 'Database 1', type: 'database' },
      { id: 3, name: 'Application 1', type: 'application' }
    ])
  })

  it('clears target CI when it matches source CI', async () => {
    await wrapper.setData({
      availableCIs: [
        { id: 1, name: 'Server 1', type: 'server' },
        { id: 2, name: 'Database 1', type: 'database' }
      ],
      selectedTargetCI: 2
    })

    await wrapper.vm.handleSourceCIChange(2)

    expect(wrapper.vm.selectedTargetCI).toBe(null)
  })

  it('loads schema attributes when schema is selected', async () => {
    await wrapper.setData({
      availableSchemas: [
        { id: 1, name: 'depends_on', attributes: [{ name: 'description', type: 'string', required: false }] }
      ]
    })

    await wrapper.vm.handleSchemaChange(1)

    expect(wrapper.vm.schemaAttributes).toEqual([
      { name: 'description', type: 'string', required: false }
    ])
  })

  it('initializes form data with default values', async () => {
    await wrapper.setData({
      availableSchemas: [
        { 
          id: 1, 
          name: 'depends_on', 
          attributes: [
            { name: 'description', type: 'string', required: false, default: 'Default description' },
            { name: 'priority', type: 'number', required: true, default: '1' }
          ]
        }
      ]
    })

    await wrapper.vm.handleSchemaChange(1)

    expect(wrapper.vm.formData).toEqual({
      description: 'Default description',
      priority: 1
    })
  })

  it('parses default values correctly based on type', () => {
    expect(wrapper.vm.parseDefaultValue('test', 'string')).toBe('test')
    expect(wrapper.vm.parseDefaultValue('25', 'number')).toBe(25)
    expect(wrapper.vm.parseDefaultValue('true', 'boolean')).toBe(true)
    expect(wrapper.vm.parseDefaultValue('["a", "b"]', 'array')).toEqual(['a', 'b'])
    expect(wrapper.vm.parseDefaultValue('{"key": "value"}', 'object')).toEqual({ key: 'value' })
  })

  it('creates relationship when form is submitted', async () => {
    await wrapper.setData({
      selectedSourceCI: 1,
      selectedTargetCI: 2,
      selectedRelationshipSchema: 1,
      availableSchemas: [
        { id: 1, name: 'depends_on', attributes: [{ name: 'description', type: 'string', required: false }] }
      ],
      schemaAttributes: [{ name: 'description', type: 'string', required: false }],
      formData: { description: 'Test relationship' }
    })

    await wrapper.vm.handleSubmit({ description: 'Test relationship' })

    expect(mockRelationshipStore.createRelationship).toHaveBeenCalledWith({
      source_id: 1,
      target_id: 2,
      schema_id: 1,
      schema_name: 'depends_on',
      description: 'Test relationship'
    })
  })

  it('navigates to relationships list after successful creation', async () => {
    const pushSpy = vi.spyOn(router, 'push')

    await wrapper.setData({
      selectedSourceCI: 1,
      selectedTargetCI: 2,
      selectedRelationshipSchema: 1,
      availableSchemas: [
        { id: 1, name: 'depends_on', attributes: [{ name: 'description', type: 'string', required: false }] }
      ],
      schemaAttributes: [{ name: 'description', type: 'string', required: false }],
      formData: { description: 'Test relationship' }
    })

    await wrapper.vm.handleSubmit({ description: 'Test relationship' })

    expect(pushSpy).toHaveBeenCalledWith('/relationships')
  })

  it('shows relationship preview when all fields are selected', async () => {
    await wrapper.setData({
      selectedSourceCI: 1,
      selectedTargetCI: 2,
      selectedRelationshipSchema: 1,
      availableCIs: [
        { id: 1, name: 'Server 1', type: 'server' },
        { id: 2, name: 'Database 1', type: 'database' }
      ],
      availableSchemas: [
        { id: 1, name: 'depends_on' }
      ]
    })

    const preview = wrapper.find('.relationship-preview')
    expect(preview.exists()).toBe(true)
    expect(preview.text()).toContain('Server 1')
    expect(preview.text()).toContain('Database 1')
    expect(preview.text()).toContain('depends_on')
  })

  it('disables create button when form is invalid', async () => {
    await wrapper.setData({
      selectedSourceCI: 1,
      selectedTargetCI: 2,
      selectedRelationshipSchema: 1,
      isFormValid: false
    })

    const createButton = wrapper.find('button[color="primary"]')
    expect(createButton.attributes('disabled')).toBe('disabled')
  })

  it('enables create button when form is valid', async () => {
    await wrapper.setData({
      selectedSourceCI: 1,
      selectedTargetCI: 2,
      selectedRelationshipSchema: 1,
      isFormValid: true
    })

    const createButton = wrapper.find('button[color="primary"]')
    expect(createButton.attributes('disabled')).toBeUndefined()
  })

  it('shows loading state during submission', async () => {
    mockRelationshipStore.createRelationship.mockImplementationOnce(() => {
      return new Promise(resolve => setTimeout(resolve, 1000))
    })

    await wrapper.setData({
      selectedSourceCI: 1,
      selectedTargetCI: 2,
      selectedRelationshipSchema: 1,
      availableSchemas: [
        { id: 1, name: 'depends_on', attributes: [{ name: 'description', type: 'string', required: false }] }
      ],
      schemaAttributes: [{ name: 'description', type: 'string', required: false }],
      formData: { description: 'Test relationship' }
    })

    await wrapper.vm.handleSubmit({ description: 'Test relationship' })

    expect(wrapper.vm.loading).toBe(true)
  })

  it('navigates back when cancel button is clicked', async () => {
    const pushSpy = vi.spyOn(router, 'push')

    await wrapper.vm.goBack()

    expect(pushSpy).toHaveBeenCalledWith('/relationships')
  })

  it('handles schema change gracefully when no schema is selected', async () => {
    await wrapper.vm.handleSchemaChange(null)

    expect(wrapper.vm.schemaAttributes).toEqual([])
    expect(Object.keys(wrapper.vm.formData)).toHaveLength(0)
  })

  it('gets CI names correctly', async () => {
    await wrapper.setData({
      availableCIs: [
        { id: 1, name: 'Server 1', type: 'server' },
        { id: 2, name: 'Database 1', type: 'database' }
      ]
    })

    expect(wrapper.vm.getSourceCIName(1)).toBe('Server 1')
    expect(wrapper.vm.getTargetCIName(2)).toBe('Database 1')
    expect(wrapper.vm.getSourceCIName(999)).toBe('Unknown')
  })

  it('gets relationship schema name correctly', async () => {
    await wrapper.setData({
      availableSchemas: [
        { id: 1, name: 'depends_on' },
        { id: 2, name: 'hosted_on' }
      ]
    })

    expect(wrapper.vm.getRelationshipSchemaName(1)).toBe('depends_on')
    expect(wrapper.vm.getRelationshipSchemaName(2)).toBe('hosted_on')
    expect(wrapper.vm.getRelationshipSchemaName(999)).toBe('Unknown')
  })

  it('shows empty state when no schema attributes are found', async () => {
    await wrapper.setData({
      selectedRelationshipSchema: 1,
      schemaAttributes: []
    })

    const emptyState = wrapper.find('.text-center.py-8')
    expect(emptyState.exists()).toBe(true)
    expect(emptyState.text()).toContain('No attributes found')
  })

  it('shows schema selection prompt when no schema is selected', async () => {
    const prompt = wrapper.find('.text-center.py-8')
    expect(prompt.exists()).toBe(true)
    expect(prompt.text()).toContain('Please select a relationship type schema')
  })
})
