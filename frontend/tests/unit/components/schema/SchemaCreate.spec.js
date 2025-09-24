import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createRouter, createWebHistory, Router } from 'vue-router'
import { createTestingPinia } from '@pinia/testing'
import SchemaCreate from '@/components/schema/SchemaCreate.vue'

// Mock stores
const mockSchemaStore = {
  createSchema: vi.fn().mockResolvedValue({
    id: 1,
    name: 'test_schema',
    type: 'ci_type',
    attributes: [
      { name: 'name', type: 'string', required: true, label: 'Name' },
      { name: 'description', type: 'string', required: false, label: 'Description' }
    ]
  }),
  updateSchema: vi.fn().mockResolvedValue({
    id: 1,
    name: 'test_schema',
    type: 'ci_type',
    attributes: [
      { name: 'name', type: 'string', required: true, label: 'Name' },
      { name: 'description', type: 'string', required: false, label: 'Description' }
    ]
  }),
  fetchSchemaDetail: vi.fn().mockResolvedValue({
    id: 1,
    name: 'test_schema',
    type: 'ci_type',
    attributes: [
      { name: 'name', type: 'string', required: true, label: 'Name' },
      { name: 'description', type: 'string', required: false, label: 'Description' }
    ]
  })
}

describe('SchemaCreate.vue', () => {
  let wrapper
  let pinia
  let router

  const createComponent = (props = {}) => {
    return mount(SchemaCreate, {
      props: {
        schemaType: 'ci_type',
        isEdit: false,
        schemaId: null,
        ...props
      },
      global: {
        plugins: [pinia, router],
        mocks: {
          $route: { params: {} },
          $router: router
        },
        provide: {
          schemaStore: mockSchemaStore
        },
        stubs: {
          'v-card': true,
          'v-card-text': true,
          'v-row': true,
          'v-col': true,
          'v-text-field': true,
          'v-select': true,
          'v-btn': true,
          'v-icon': true,
          'v-divider': true,
          'v-chip': true,
          'v-alert': true,
          'v-form': true,
          'v-textarea': true,
          'v-switch': true
        }
      }
    })
  }

  beforeEach(() => {
    vi.clearAllMocks()
    
    pinia = createTestingPinia({
      createSpy: vi.fn,
      initialState: {
        schema: mockSchemaStore
      }
    })

    router = createRouter({
      history: createWebHistory(),
      routes: [
        { path: '/schemas', name: 'schemas' },
        { path: '/schemas/create', name: 'schema-create' }
      ]
    })

    wrapper = createComponent()
  })

  afterEach(() => {
    wrapper.unmount()
  })

  it('renders properly in create mode', () => {
    expect(wrapper.exists()).toBe(true)
    expect(wrapper.find('h1').text()).toBe('Create Schema')
  })

  it('renders properly in edit mode', async () => {
    const editWrapper = createComponent({
      isEdit: true,
      schemaId: 1
    })

    await editWrapper.vm.$nextTick()
    
    expect(editWrapper.exists()).toBe(true)
    expect(editWrapper.find('h1').text()).toBe('Edit Schema')
  })

  it('loads schema data in edit mode', async () => {
    const editWrapper = createComponent({
      isEdit: true,
      schemaId: 1
    })

    await editWrapper.vm.$nextTick()
    
    expect(mockSchemaStore.fetchSchemaDetail).toHaveBeenCalledWith(1)
  })

  it('adds new attribute row', async () => {
    await wrapper.vm.addAttribute()
    
    expect(wrapper.vm.attributes).toHaveLength(1)
    expect(wrapper.vm.attributes[0]).toEqual({
      name: '',
      type: 'string',
      required: false,
      label: '',
      description: '',
      default: ''
    })
  })

  it('removes attribute row', async () => {
    // Add some attributes first
    await wrapper.setData({
      attributes: [
        { name: 'name', type: 'string', required: true, label: 'Name' },
        { name: 'description', type: 'string', required: false, label: 'Description' }
      ]
    })

    await wrapper.vm.removeAttribute(0)
    
    expect(wrapper.vm.attributes).toHaveLength(1)
    expect(wrapper.vm.attributes[0].name).toBe('description')
  })

  it('validates required fields', async () => {
    await wrapper.setData({
      schemaName: '',
      attributes: [
        { name: '', type: 'string', required: true, label: '' }
      ]
    })

    const isValid = await wrapper.vm.validateForm()
    
    expect(isValid).toBe(false)
    expect(wrapper.vm.errors).toContain('Schema name is required')
    expect(wrapper.vm.errors).toContain('Attribute name is required')
    expect(wrapper.vm.errors).toContain('Attribute label is required')
  })

  it('validates attribute names for uniqueness', async () => {
    await wrapper.setData({
      schemaName: 'test_schema',
      attributes: [
        { name: 'name', type: 'string', required: true, label: 'Name' },
        { name: 'name', type: 'string', required: false, label: 'Name 2' }
      ]
    })

    const isValid = await wrapper.vm.validateForm()
    
    expect(isValid).toBe(false)
    expect(wrapper.vm.errors).toContain('Attribute names must be unique')
  })

  it('creates schema successfully', async () => {
    await wrapper.setData({
      schemaName: 'test_schema',
      attributes: [
        { name: 'name', type: 'string', required: true, label: 'Name' },
        { name: 'description', type: 'string', required: false, label: 'Description' }
      ]
    })

    await wrapper.vm.handleSubmit()
    
    expect(mockSchemaStore.createSchema).toHaveBeenCalledWith({
      name: 'test_schema',
      type: 'ci_type',
      attributes: [
        { name: 'name', type: 'string', required: true, label: 'Name' },
        { name: 'description', type: 'string', required: false, label: 'Description' }
      ]
    })
  })

  it('updates schema successfully in edit mode', async () => {
    const editWrapper = createComponent({
      isEdit: true,
      schemaId: 1
    })

    await editWrapper.setData({
      schemaName: 'updated_schema',
      attributes: [
        { name: 'name', type: 'string', required: true, label: 'Name' },
        { name: 'description', type: 'string', required: false, label: 'Description' }
      ]
    })

    await editWrapper.vm.handleSubmit()
    
    expect(mockSchemaStore.updateSchema).toHaveBeenCalledWith({
      id: 1,
      name: 'updated_schema',
      type: 'ci_type',
      attributes: [
        { name: 'name', type: 'string', required: true, label: 'Name' },
        { name: 'description', type: 'string', required: false, label: 'Description' }
      ]
    })
  })

  it('navigates back on cancel', async () => {
    const pushSpy = vi.spyOn(router, 'push')

    await wrapper.vm.cancel()
    
    expect(pushSpy).toHaveBeenCalledWith('/schemas')
  })

  it('shows loading state during submission', async () => {
    mockSchemaStore.createSchema.mockImplementationOnce(() => {
      return new Promise(resolve => setTimeout(resolve, 1000))
    })

    await wrapper.setData({
      schemaName: 'test_schema',
      attributes: [
        { name: 'name', type: 'string', required: true, label: 'Name' }
      ]
    })

    await wrapper.vm.handleSubmit()
    
    expect(wrapper.vm.loading).toBe(true)
  })

  it('handles API errors gracefully', async () => {
    mockSchemaStore.createSchema.mockRejectedValueOnce(new Error('API Error'))

    await wrapper.setData({
      schemaName: 'test_schema',
      attributes: [
        { name: 'name', type: 'string', required: true, label: 'Name' }
      ]
    })

    await wrapper.vm.handleSubmit()
    
    expect(wrapper.vm.loading).toBe(false)
    expect(wrapper.vm.errors).toContain('Failed to create schema: API Error')
  })

  it('formats attribute names correctly', () => {
    expect(wrapper.vm.formatAttributeName('test_name')).toBe('Test Name')
    expect(wrapper.vm.formatAttributeName('another_test_name')).toBe('Another Test Name')
  })

  it('gets correct field icons based on type', () => {
    expect(wrapper.vm.getFieldIcon('string')).toBe('mdi-text')
    expect(wrapper.vm.getFieldIcon('number')).toBe('mdi-numeric')
    expect(wrapper.vm.getFieldIcon('boolean')).toBe('mdi-toggle-switch')
    expect(wrapper.vm.getFieldIcon('date')).toBe('mdi-calendar')
    expect(wrapper.vm.getFieldIcon('array')).toBe('mdi-array')
    expect(wrapper.vm.getFieldIcon('object')).toBe('mdi-code-braces')
  })

  it('shows attribute type options', () => {
    const typeOptions = wrapper.vm.attributeTypeOptions
    expect(typeOptions).toEqual([
      { title: 'String', value: 'string' },
      { title: 'Number', value: 'number' },
      { title: 'Boolean', value: 'boolean' },
      { title: 'Date', value: 'date' },
      { title: 'Array', value: 'array' },
      { title: 'Object', value: 'object' }
    ])
  })

  it('disables remove button for single attribute', async () => {
    await wrapper.setData({
      attributes: [
        { name: 'name', type: 'string', required: true, label: 'Name' }
      ]
    })

    const removeButton = wrapper.find('button[color="error"]')
    expect(removeButton.attributes('disabled')).toBe('disabled')
  })

  it('enables remove button for multiple attributes', async () => {
    await wrapper.setData({
      attributes: [
        { name: 'name', type: 'string', required: true, label: 'Name' },
        { name: 'description', type: 'string', required: false, label: 'Description' }
      ]
    })

    const removeButtons = wrapper.findAll('button[color="error"]')
    expect(removeButtons[0].attributes('disabled')).toBeUndefined()
    expect(removeButtons[1].attributes('disabled')).toBeUndefined()
  })

  it('shows schema type in title', async () => {
    const ciTypeWrapper = createComponent({ schemaType: 'ci_type' })
    expect(ciTypeWrapper.find('h1').text()).toBe('Create CI Type Schema')

    const relationshipTypeWrapper = createComponent({ schemaType: 'relationship_type' })
    expect(relationshipTypeWrapper.find('h1').text()).toBe('Create Relationship Type Schema')
  })

  it('handles empty attributes array gracefully', () => {
    expect(wrapper.vm.attributes).toEqual([])
    expect(wrapper.vm.canRemoveAttribute).toBe(false)
  })

  it('validates attribute labels are not empty', async () => {
    await wrapper.setData({
      schemaName: 'test_schema',
      attributes: [
        { name: 'name', type: 'string', required: true, label: '' }
      ]
    })

    const isValid = await wrapper.vm.validateForm()
    
    expect(isValid).toBe(false)
    expect(wrapper.vm.errors).toContain('Attribute label is required')
  })
})
