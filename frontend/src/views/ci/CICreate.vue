<template>
  <div class="ci-create-container">
    <div class="page-header">
      <h1>Create Configuration Item</h1>
      <v-btn @click="goBack" variant="text">
        <v-icon left>mdi-arrow-left</v-icon>
        Back to CIs
      </v-btn>
    </div>

    <v-card class="form-card">
      <v-card-text>
        <!-- Schema Selection -->
        <v-row class="mb-6">
          <v-col cols="12">
            <v-select
              v-model="selectedSchema"
              :items="availableSchemas"
              label="Select CI Type Schema *"
              item-title="name"
              item-value="id"
              :rules="[rules.required]"
              @update:model-value="handleSchemaChange"
              prepend-inner-icon="mdi-database-outline"
            />
          </v-col>
        </v-row>

        <!-- Dynamic Form -->
        <div v-if="selectedSchema && schemaAttributes.length > 0">
          <DynamicForm
            :schema-attributes="schemaAttributes"
            :initial-data="formData"
            :schema-type="'ci_type'"
            :enable-real-time-validation="true"
            @submit="handleSubmit"
            @validation-change="handleValidationChange"
            @data-change="handleDataChange"
          >
            <template #actions>
              <div class="form-actions">
                <v-btn @click="goBack" variant="outlined">Cancel</v-btn>
                <v-btn
                  color="primary"
                  type="submit"
                  :loading="loading"
                  :disabled="!isFormValid"
                >
                  <v-icon left>mdi-check</v-icon>
                  Create CI
                </v-btn>
              </div>
            </template>
          </DynamicForm>
        </div>

        <!-- Empty State -->
        <div v-else-if="selectedSchema && schemaAttributes.length === 0" class="text-center py-8">
          <v-icon large color="grey">mdi-alert-circle-outline</v-icon>
          <p class="text-grey mt-2">No attributes found for the selected schema. Please select a different schema or add attributes to the schema.</p>
        </div>

        <!-- Schema Selection Prompt -->
        <div v-else class="text-center py-8">
          <v-icon large color="grey">mdi-database-outline</v-icon>
          <p class="text-grey mt-2">Please select a CI type schema to continue creating the configuration item.</p>
        </div>
      </v-card-text>
    </v-card>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useSchemaStore } from '@/stores/schema'
import { useCIStore } from '@/stores/ci'
import DynamicForm from '@/components/forms/DynamicForm.vue'

const router = useRouter()
const schemaStore = useSchemaStore()
const ciStore = useCIStore()

// Form state
const formRef = ref()
const loading = ref(false)
const selectedSchema = ref(null)
const availableSchemas = ref([])
const schemaAttributes = ref([])
const isFormValid = ref(false)

// Form data
const formData = reactive({})

// Validation rules
const rules = {
  required: (v) => !!v || 'This field is required'
}

// Computed properties
const isFormReady = computed(() => {
  return selectedSchema.value && schemaAttributes.value.length > 0
})

// Methods
const loadAvailableSchemas = async () => {
  try {
    const response = await schemaStore.fetchCiTypeSchemas({
      page: 1,
      page_size: 100,
      is_active: true
    })
    availableSchemas.value = response.schemas || []
  } catch (error) {
    console.error('Failed to load available schemas:', error)
  }
}

const handleSchemaChange = async (schemaId) => {
  if (!schemaId) {
    schemaAttributes.value = []
    // Clear form data
    Object.keys(formData).forEach(key => {
      delete formData[key]
    })
    return
  }

  try {
    // Load schema details
    const schema = availableSchemas.value.find(s => s.id === schemaId)
    if (schema && schema.attributes) {
      schemaAttributes.value = schema.attributes
      
      // Initialize form data with default values
      Object.keys(formData).forEach(key => {
        delete formData[key]
      })
      
      schema.attributes.forEach(attr => {
        if (attr.default !== undefined && attr.default !== '') {
          formData[attr.name] = parseDefaultValue(attr.default, attr.type)
        } else {
          // Set default based on type
          switch (attr.type) {
            case 'boolean':
              formData[attr.name] = false
              break
            case 'array':
              formData[attr.name] = []
              break
            case 'object':
              formData[attr.name] = {}
              break
            default:
              formData[attr.name] = ''
          }
        }
      })
    }
  } catch (error) {
    console.error('Failed to load schema details:', error)
    schemaAttributes.value = []
  }
}

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

const handleSubmit = async (data) => {
  if (!selectedSchema.value) {
    console.warn('No schema selected')
    return
  }

  try {
    loading.value = true
    
    // Prepare CI data
    const ciData = {
      ...data,
      schema_id: selectedSchema.value,
      schema_name: availableSchemas.value.find(s => s.id === selectedSchema.value)?.name
    }
    
    // Create CI using the CI store
    await ciStore.createCI(ciData)
    
    // Navigate back to CI list
    router.push('/cis')
  } catch (error) {
    console.error('Failed to create CI:', error)
  } finally {
    loading.value = false
  }
}

const handleValidationChange = (validationData) => {
  // Update form validation state
  isFormValid.value = validationData.isValid
}

const handleDataChange = (data) => {
  // Update form data
  Object.assign(formData, data)
}

const goBack = () => {
  router.push('/cis')
}

// Lifecycle
onMounted(() => {
  loadAvailableSchemas()
})

// Watch for schema changes
watch(selectedSchema, (newSchemaId) => {
  if (newSchemaId) {
    handleSchemaChange(newSchemaId)
  }
})
</script>

<style scoped>
.ci-create-container {
  padding: 20px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.page-header h1 {
  margin: 0;
  font-size: 24px;
  color: #303133;
}

.form-card {
  max-width: 1200px;
  margin: 0 auto;
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  margin-top: 30px;
  padding-top: 20px;
  border-top: 1px solid #e0e0e0;
}

.v-card {
  border-radius: 8px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.v-card-text {
  padding: 24px;
}
</style>
