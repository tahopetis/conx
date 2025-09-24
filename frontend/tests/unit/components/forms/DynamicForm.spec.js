import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import DynamicForm from '@/components/forms/DynamicForm.vue'
import { createTestingPinia } from '@pinia/testing'

describe('DynamicForm.vue', () => {
  let wrapper
  let pinia

  const mockSchemaAttributes = [
    {
      name: 'name',
      type: 'string',
      label: 'Name',
      required: true,
      description: 'The name of the item'
    },
    {
      name: 'age',
      type: 'number',
      label: 'Age',
      required: false,
      default: '25'
    },
    {
      name: 'active',
      type: 'boolean',
      label: 'Active',
      required: false,
      default: 'true'
    },
    {
      name: 'tags',
      type: 'array',
      label: 'Tags',
      required: false
    },
    {
      name: 'metadata',
      type: 'object',
      label: 'Metadata',
      required: false
    },
    {
      name: 'created_at',
      type: 'date',
      label: 'Created Date',
      required: false
    }
  ]

  const mockInitialData = {
    name: 'Test Item',
    age: 30,
    active: true,
    tags: ['tag1', 'tag2'],
    metadata: { key: 'value' },
    created_at: '2023-01-01T00:00:00Z'
  }

  beforeEach(() => {
    pinia = createTestingPinia()
    wrapper = mount(DynamicForm, {
      props: {
        schemaAttributes: mockSchemaAttributes,
        initialData: mockInitialData,
        schemaType: 'ci_type',
        enableRealTimeValidation: true
      },
      global: {
        plugins: [pinia],
        stubs: {
          'v-form': true,
          'v-text-field': true,
          'v-number-input': true,
          'v-checkbox': true,
          'v-select': true,
          'v-date-picker': true,
          'v-textarea': true,
          'v-btn': true,
          'v-card': true,
          'v-card-text': true,
          'v-alert': true
        }
      }
    })
  })

  afterEach(() => {
    wrapper.unmount()
  })

  it('renders properly with schema attributes', () => {
    expect(wrapper.exists()).toBe(true)
    expect(wrapper.text()).toContain('Name')
    expect(wrapper.text()).toContain('Age')
    expect(wrapper.text()).toContain('Active')
    expect(wrapper.text()).toContain('Tags')
    expect(wrapper.text()).toContain('Metadata')
    expect(wrapper.text()).toContain('Created Date')
  })

  it('initializes form with initial data', async () => {
    await wrapper.vm.$nextTick()
    
    const formData = wrapper.vm.formData
    expect(formData.name).toBe('Test Item')
    expect(formData.age).toBe(30)
    expect(formData.active).toBe(true)
    expect(formData.tags).toEqual(['tag1', 'tag2'])
    expect(formData.metadata).toEqual({ key: 'value' })
  })

  it('applies default values for missing data', async () => {
    const wrapperWithDefaults = mount(DynamicForm, {
      props: {
        schemaAttributes: mockSchemaAttributes,
        initialData: { name: 'Test' },
        schemaType: 'ci_type',
        enableRealTimeValidation: true
      },
      global: {
        plugins: [pinia],
        stubs: {
          'v-form': true,
          'v-text-field': true,
          'v-number-input': true,
          'v-checkbox': true,
          'v-select': true,
          'v-date-picker': true,
          'v-textarea': true,
          'v-btn': true,
          'v-card': true,
          'v-card-text': true,
          'v-alert': true
        }
      }
    })

    await wrapperWithDefaults.vm.$nextTick()
    
    const formData = wrapperWithDefaults.vm.formData
    expect(formData.name).toBe('Test')
    expect(formData.age).toBe(25) // default value
    expect(formData.active).toBe(true) // default value
    expect(formData.tags).toEqual([]) // default for array
    expect(formData.metadata).toEqual({}) // default for object
  })

  it('emits validation-change event on form validation', async () => {
    const validationChangeSpy = vi.fn()
    wrapper.vm.$on('validation-change', validationChangeSpy)

    // Simulate form validation
    await wrapper.vm.validateForm()

    expect(validationChangeSpy).toHaveBeenCalled()
    const validationData = validationChangeSpy.mock.calls[0][0]
    expect(validationData).toHaveProperty('isValid')
    expect(validationData).toHaveProperty('errors')
  })

  it('emits data-change event when form data changes', async () => {
    const dataChangeSpy = vi.fn()
    wrapper.vm.$on('data-change', dataChangeSpy)

    // Simulate data change
    await wrapper.vm.handleDataChange({ name: 'Updated Name' })

    expect(dataChangeSpy).toHaveBeenCalled()
    const data = dataChangeSpy.mock.calls[0][0]
    expect(data.name).toBe('Updated Name')
  })

  it('emits submit event when form is submitted', async () => {
    const submitSpy = vi.fn()
    wrapper.vm.$on('submit', submitSpy)

    // Simulate form submission
    await wrapper.vm.handleSubmit(mockInitialData)

    expect(submitSpy).toHaveBeenCalled()
    const submittedData = submitSpy.mock.calls[0][0]
    expect(submittedData).toEqual(mockInitialData)
  })

  it('validates required fields correctly', async () => {
    const validationChangeSpy = vi.fn()
    wrapper.vm.$on('validation-change', validationChangeSpy)

    // Test with empty required field
    await wrapper.vm.validateForm({ name: '' })

    expect(validationChangeSpy).toHaveBeenCalled()
    const validationData = validationChangeSpy.mock.calls[0][0]
    expect(validationData.isValid).toBe(false)
    expect(validationData.errors).toContain('Name is required')
  })

  it('handles different data types correctly', () => {
    const formData = wrapper.vm.formData
    
    // String type
    expect(typeof formData.name).toBe('string')
    
    // Number type
    expect(typeof formData.age).toBe('number')
    
    // Boolean type
    expect(typeof formData.active).toBe('boolean')
    
    // Array type
    expect(Array.isArray(formData.tags)).toBe(true)
    
    // Object type
    expect(typeof formData.metadata).toBe('object')
    expect(Array.isArray(formData.metadata)).toBe(false)
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

  it('renders field descriptions when available', () => {
    expect(wrapper.text()).toContain('The name of the item')
  })

  it('marks required fields with asterisk', () => {
    const nameField = wrapper.find('[data-testid="field-name"]')
    expect(nameField.exists()).toBe(true)
    // The required field should have an asterisk indicator
  })

  it('handles real-time validation when enabled', async () => {
    const validationChangeSpy = vi.fn()
    wrapper.vm.$on('validation-change', validationChangeSpy)

    // Simulate real-time validation
    await wrapper.vm.handleFieldChange('name', '')

    expect(validationChangeSpy).toHaveBeenCalled()
  })

  it('does not perform real-time validation when disabled', async () => {
    const wrapperWithNoValidation = mount(DynamicForm, {
      props: {
        schemaAttributes: mockSchemaAttributes,
        initialData: mockInitialData,
        schemaType: 'ci_type',
        enableRealTimeValidation: false
      },
      global: {
        plugins: [pinia],
        stubs: {
          'v-form': true,
          'v-text-field': true,
          'v-number-input': true,
          'v-checkbox': true,
          'v-select': true,
          'v-date-picker': true,
          'v-textarea': true,
          'v-btn': true,
          'v-card': true,
          'v-card-text': true,
          'v-alert': true
        }
      }
    })

    const validationChangeSpy = vi.fn()
    wrapperWithNoValidation.vm.$on('validation-change', validationChangeSpy)

    // Simulate field change
    await wrapperWithNoValidation.vm.handleFieldChange('name', '')

    // Should not trigger validation when disabled
    expect(validationChangeSpy).not.toHaveBeenCalled()
  })

  it('handles empty schema attributes gracefully', () => {
    const wrapperWithEmptySchema = mount(DynamicForm, {
      props: {
        schemaAttributes: [],
        initialData: {},
        schemaType: 'ci_type',
        enableRealTimeValidation: true
      },
      global: {
        plugins: [pinia],
        stubs: {
          'v-form': true,
          'v-text-field': true,
          'v-number-input': true,
          'v-checkbox': true,
          'v-select': true,
          'v-date-picker': true,
          'v-textarea': true,
          'v-btn': true,
          'v-card': true,
          'v-card-text': true,
          'v-alert': true
        }
      }
    })

    expect(wrapperWithEmptySchema.exists()).toBe(true)
    expect(wrapperWithEmptySchema.vm.formData).toEqual({})
  })

  it('parses default values correctly based on type', () => {
    const stringDefault = wrapper.vm.parseDefaultValue('test', 'string')
    expect(stringDefault).toBe('test')

    const numberDefault = wrapper.vm.parseDefaultValue('25', 'number')
    expect(numberDefault).toBe(25)

    const booleanDefault = wrapper.vm.parseDefaultValue('true', 'boolean')
    expect(booleanDefault).toBe(true)

    const arrayDefault = wrapper.vm.parseDefaultValue('["a", "b"]', 'array')
    expect(arrayDefault).toEqual(['a', 'b'])

    const objectDefault = wrapper.vm.parseDefaultValue('{"key": "value"}', 'object')
    expect(objectDefault).toEqual({ key: 'value' })
  })
})
