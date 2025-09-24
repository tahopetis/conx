<template>
  <div class="dynamic-form">
    <v-form ref="form" v-model="valid" @submit.prevent="$emit('submit', formData)">
      <div v-for="attribute in schemaAttributes" :key="attribute.name" class="form-field mb-4">
        <!-- String Field -->
        <v-text-field
          v-if="attribute.type === 'string'"
          v-model="formData[attribute.name]"
          :label="getAttributeLabel(attribute)"
          :placeholder="getAttributePlaceholder(attribute)"
          :rules="getFieldRules(attribute)"
          :required="attribute.required"
          :hint="attribute.description"
          persistent-hint
          @input="validateField(attribute.name, $event)"
        />

        <!-- Number Field -->
        <v-text-field
          v-else-if="attribute.type === 'number'"
          v-model.number="formData[attribute.name]"
          :label="getAttributeLabel(attribute)"
          :placeholder="getAttributePlaceholder(attribute)"
          :rules="getFieldRules(attribute)"
          :required="attribute.required"
          :hint="attribute.description"
          persistent-hint
          type="number"
          @input="validateField(attribute.name, $event)"
        />

        <!-- Boolean Field -->
        <v-switch
          v-else-if="attribute.type === 'boolean'"
          v-model="formData[attribute.name]"
          :label="getAttributeLabel(attribute)"
          :hint="attribute.description"
          persistent-hint
          inset
          @change="validateField(attribute.name, $event)"
        />

        <!-- Date Field -->
        <v-text-field
          v-else-if="attribute.type === 'date'"
          v-model="formData[attribute.name]"
          :label="getAttributeLabel(attribute)"
          :placeholder="getAttributePlaceholder(attribute)"
          :rules="getFieldRules(attribute)"
          :required="attribute.required"
          :hint="attribute.description"
          persistent-hint
          type="date"
          @input="validateField(attribute.name, $event)"
        />

        <!-- Array Field -->
        <v-combobox
          v-else-if="attribute.type === 'array'"
          v-model="formData[attribute.name]"
          :label="getAttributeLabel(attribute)"
          :placeholder="getAttributePlaceholder(attribute)"
          :rules="getFieldRules(attribute)"
          :required="attribute.required"
          :hint="attribute.description"
          persistent-hint
          multiple
          chips
          small-chips
          deletable-chips
          @input="validateField(attribute.name, $event)"
        />

        <!-- Object Field -->
        <v-textarea
          v-else-if="attribute.type === 'object'"
          v-model="formData[attribute.name]"
          :label="getAttributeLabel(attribute)"
          :placeholder="getAttributePlaceholder(attribute)"
          :rules="getFieldRules(attribute)"
          :required="attribute.required"
          :hint="attribute.description"
          persistent-hint
          rows="3"
          @input="validateField(attribute.name, $event)"
        />

        <!-- Field Error Display -->
        <div v-if="fieldErrors[attribute.name]" class="error-messages mt-1">
          <div v-for="(error, index) in fieldErrors[attribute.name]" :key="index" class="error-message">
            <v-icon small color="error" class="mr-1">mdi-alert-circle</v-icon>
            {{ error }}
          </div>
        </div>
      </div>

      <!-- Form Actions Slot -->
      <slot name="actions"></slot>
    </v-form>
  </div>
</template>

<script setup>
import { ref, reactive, computed, watch, onMounted } from 'vue'
import { useSchemaStore } from '@/stores/schema'

const props = defineProps({
  // Schema attributes array
  schemaAttributes: {
    type: Array,
    required: true,
    default: () => []
  },
  // Initial form data
  initialData: {
    type: Object,
    default: () => ({})
  },
  // Schema type for validation
  schemaType: {
    type: String,
    default: 'ci_type'
  },
  // Enable real-time validation
  enableRealTimeValidation: {
    type: Boolean,
    default: true
  }
})

const emit = defineEmits(['submit', 'validation-change', 'data-change'])

const schemaStore = useSchemaStore()
const form = ref(null)
const valid = ref(false)

// Form data
const formData = reactive({})

// Field errors
const fieldErrors = reactive({})

// Validation state
const isValid = computed(() => valid.value && Object.keys(fieldErrors).length === 0)

// Initialize form data
const initializeFormData = () => {
  // Clear existing data
  Object.keys(formData).forEach(key => {
    delete formData[key]
  })
  
  // Apply default values from schema
  props.schemaAttributes.forEach(attribute => {
    if (attribute.default !== undefined && attribute.default !== '') {
      formData[attribute.name] = parseDefaultValue(attribute.default, attribute.type)
    } else if (props.initialData[attribute.name] !== undefined) {
      formData[attribute.name] = props.initialData[attribute.name]
    } else {
      // Set default based on type
      switch (attribute.type) {
        case 'boolean':
          formData[attribute.name] = false
          break
        case 'array':
          formData[attribute.name] = []
          break
        case 'object':
          formData[attribute.name] = {}
          break
        default:
          formData[attribute.name] = ''
      }
    }
  })
}

// Parse default value based on type
const parseDefaultValue = (value, type) => {
  if (value === undefined || value === '') return value
  
  try {
    switch (type) {
      case 'number':
        return Number(value)
      case 'boolean':
        return value === 'true' || value === true
      case 'array':
        return Array.isArray(value) ? value : JSON.parse(value)
      case 'object':
        return typeof value === 'object' ? value : JSON.parse(value)
      default:
        return String(value)
    }
  } catch (error) {
    console.warn(`Failed to parse default value "${value}" for type ${type}:`, error)
    return value
  }
}

// Get attribute label
const getAttributeLabel = (attribute) => {
  return attribute.name.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase()) + 
         (attribute.required ? ' *' : '')
}

// Get attribute placeholder
const getAttributePlaceholder = (attribute) => {
  const placeholders = {
    string: 'Enter text value',
    number: 'Enter number',
    boolean: 'Select true or false',
    date: 'YYYY-MM-DD',
    array: 'Enter values (comma separated)',
    object: 'Enter JSON object'
  }
  return placeholders[attribute.type] || 'Enter value'
}

// Get field validation rules
const getFieldRules = (attribute) => {
  const rules = []
  
  // Required rule
  if (attribute.required) {
    rules.push(v => {
      if (v === undefined || v === null || v === '') {
        return `${getAttributeLabel(attribute)} is required`
      }
      if (Array.isArray(v) && v.length === 0) {
        return `${getAttributeLabel(attribute)} is required`
      }
      return true
    })
  }
  
  // Type-specific rules
  if (attribute.validation) {
    const { validation } = attribute
    
    // String validation
    if (attribute.type === 'string') {
      if (validation.minLength) {
        rules.push(v => !v || v.length >= validation.minLength || 
          `Minimum length is ${validation.minLength} characters`)
      }
      if (validation.maxLength) {
        rules.push(v => !v || v.length <= validation.maxLength || 
          `Maximum length is ${validation.maxLength} characters`)
      }
      if (validation.pattern) {
        rules.push(v => !v || new RegExp(validation.pattern).test(v) || 
          `Value must match pattern: ${validation.pattern}`)
      }
      if (validation.format) {
        rules.push(v => validateFormat(v, validation.format))
      }
      if (validation.enum) {
        rules.push(v => !v || validation.enum.includes(v) || 
          `Value must be one of: ${validation.enum.join(', ')}`)
      }
    }
    
    // Number validation
    if (attribute.type === 'number') {
      if (validation.min !== undefined) {
        rules.push(v => v === undefined || v === null || v >= validation.min || 
          `Minimum value is ${validation.min}`)
      }
      if (validation.max !== undefined) {
        rules.push(v => v === undefined || v === null || v <= validation.max || 
          `Maximum value is ${validation.max}`)
      }
    }
    
    // Array validation
    if (attribute.type === 'array') {
      if (validation.minLength) {
        rules.push(v => !v || v.length >= validation.minLength || 
          `Minimum ${validation.minLength} items required`)
      }
      if (validation.maxLength) {
        rules.push(v => !v || v.length <= validation.maxLength || 
          `Maximum ${validation.maxLength} items allowed`)
      }
    }
  }
  
  return rules
}

// Validate format
const validateFormat = (value, format) => {
  if (!value) return true
  
  const formatValidators = {
    email: (v) => /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(v) || 'Invalid email format',
    ipv4: (v) => /^(\d{1,3}\.){3}\d{1,3}$/.test(v) || 'Invalid IPv4 address',
    ipv6: (v) => /^([0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}$/.test(v) || 'Invalid IPv6 address',
    url: (v) => {
      try {
        new URL(v)
        return true
      } catch {
        return 'Invalid URL format'
      }
    },
    uuid: (v) => /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i.test(v) || 'Invalid UUID format'
  }
  
  return formatValidators[format]?.(value) || true
}

// Validate individual field
const validateField = async (fieldName, value) => {
  // Clear previous errors for this field
  delete fieldErrors[fieldName]
  
  if (!props.enableRealTimeValidation) {
    return
  }
  
  // Find the attribute definition
  const attribute = props.schemaAttributes.find(attr => attr.name === fieldName)
  if (!attribute) return
  
  // Validate against schema
  try {
    const validationData = {
      [fieldName]: value
    }
    
    let result
    if (props.schemaType === 'ci_type') {
      result = await schemaStore.validateCIAgainstSchema({
        ciData: validationData,
        schemaData: { attributes: [attribute] }
      })
    } else {
      result = await schemaStore.validateRelationshipAgainstSchema({
        relationshipData: validationData,
        schemaData: { attributes: [attribute] }
      })
    }
    
    // Set field errors if validation failed
    if (!result.isValid && result.errors) {
      fieldErrors[fieldName] = result.errors
        .filter(error => error.field === fieldName)
        .map(error => error.message)
    }
  } catch (error) {
    console.error('Field validation failed:', error)
  }
  
  // Emit validation change event
  emit('validation-change', {
    fieldName,
    isValid: !fieldErrors[fieldName] || fieldErrors[fieldName].length === 0,
    errors: fieldErrors[fieldName] || []
  })
}

// Validate entire form
const validateForm = async () => {
  if (!form.value) return false
  
  // Validate Vue form rules
  const isValid = await form.value.validate()
  
  if (!isValid) return false
  
  // Validate against schema
  try {
    let result
    if (props.schemaType === 'ci_type') {
      result = await schemaStore.validateCIAgainstSchema({
        ciData: formData,
        schemaData: { attributes: props.schemaAttributes }
      })
    } else {
      result = await schemaStore.validateRelationshipAgainstSchema({
        relationshipData: formData,
        schemaData: { attributes: props.schemaAttributes }
      })
    }
    
    // Set field errors
    Object.keys(fieldErrors).forEach(key => delete fieldErrors[key])
    
    if (!result.isValid && result.errors) {
      result.errors.forEach(error => {
        if (!fieldErrors[error.field]) {
          fieldErrors[error.field] = []
        }
        fieldErrors[error.field].push(error.message)
      })
    }
    
    return result.isValid
  } catch (error) {
    console.error('Form validation failed:', error)
    return false
  }
}

// Reset form
const resetForm = () => {
  if (form.value) {
    form.value.reset()
  }
  initializeFormData()
  Object.keys(fieldErrors).forEach(key => delete fieldErrors[key])
}

// Watch for schema changes
watch(() => props.schemaAttributes, initializeFormData, { deep: true })

// Watch for initial data changes
watch(() => props.initialData, initializeFormData, { deep: true })

// Watch form data changes
watch(formData, (newData) => {
  emit('data-change', { ...newData })
}, { deep: true })

// Initialize on mount
onMounted(() => {
  initializeFormData()
})

defineExpose({
  validateForm,
  resetForm,
  isValid,
  formData
})
</script>

<style scoped>
.dynamic-form {
  width: 100%;
}

.form-field {
  position: relative;
}

.error-messages {
  color: #b71c1c;
  font-size: 0.75rem;
  margin-top: 4px;
}

.error-message {
  display: flex;
  align-items: center;
  margin-bottom: 2px;
}
</style>
