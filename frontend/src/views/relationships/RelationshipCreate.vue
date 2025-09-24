<template>
  <div class="relationship-create-container">
    <div class="page-header">
      <h1>Create Relationship</h1>
      <v-btn @click="goBack" variant="text">
        <v-icon left>mdi-arrow-left</v-icon>
        Back to Relationships
      </v-btn>
    </div>

    <v-card class="form-card">
      <v-card-text>
        <!-- Basic Relationship Information -->
        <v-row class="mb-6">
          <v-col cols="12" md="6">
            <v-select
              v-model="selectedSourceCI"
              :items="availableCIs"
              label="Source Configuration Item *"
              item-title="name"
              item-value="id"
              :rules="[rules.required]"
              @update:model-value="handleSourceCIChange"
              prepend-inner-icon="mdi-server"
              clearable
            />
          </v-col>
          <v-col cols="12" md="6">
            <v-select
              v-model="selectedTargetCI"
              :items="availableTargetCIs"
              label="Target Configuration Item *"
              item-title="name"
              item-value="id"
              :rules="[rules.required]"
              @update:model-value="handleTargetCIChange"
              prepend-inner-icon="mdi-server-network"
              clearable
            />
          </v-col>
        </v-row>

        <v-row class="mb-6">
          <v-col cols="12">
            <v-select
              v-model="selectedRelationshipSchema"
              :items="availableSchemas"
              label="Relationship Type Schema *"
              item-title="name"
              item-value="id"
              :rules="[rules.required]"
              @update:model-value="handleSchemaChange"
              prepend-inner-icon="mdi-link-variant"
              clearable
            />
          </v-col>
        </v-row>

        <!-- Dynamic Form for Relationship Attributes -->
        <div v-if="selectedRelationshipSchema && schemaAttributes.length > 0">
          <DynamicForm
            :schema-attributes="schemaAttributes"
            :initial-data="formData"
            :schema-type="'relationship_type'"
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
                  :disabled="!isFormValid || !selectedSourceCI || !selectedTargetCI"
                >
                  <v-icon left>mdi-check</v-icon>
                  Create Relationship
                </v-btn>
              </div>
            </template>
          </DynamicForm>
        </div>

        <!-- Empty State -->
        <div v-else-if="selectedRelationshipSchema && schemaAttributes.length === 0" class="text-center py-8">
          <v-icon large color="grey">mdi-alert-circle-outline</v-icon>
          <p class="text-grey mt-2">No attributes found for the selected relationship schema. Please select a different schema or add attributes to the schema.</p>
        </div>

        <!-- Schema Selection Prompt -->
        <div v-else class="text-center py-8">
          <v-icon large color="grey">mdi-link-variant</v-icon>
          <p class="text-grey mt-2">Please select a relationship type schema to continue creating the relationship.</p>
        </div>

        <!-- Relationship Preview -->
        <div v-if="selectedSourceCI && selectedTargetCI && selectedRelationshipSchema" class="relationship-preview mt-6">
          <v-divider class="mb-4"></v-divider>
          <h4>Relationship Preview</h4>
          <div class="preview-content">
            <v-chip color="primary" class="source-chip">
              {{ getSourceCIName(selectedSourceCI) }}
            </v-chip>
            <v-icon class="arrow-icon">mdi-arrow-right</v-icon>
            <v-chip color="secondary" class="relationship-chip">
              {{ getRelationshipSchemaName(selectedRelationshipSchema) }}
            </v-chip>
            <v-icon class="arrow-icon">mdi-arrow-right</v-icon>
            <v-chip color="success" class="target-chip">
              {{ getTargetCIName(selectedTargetCI) }}
            </v-chip>
          </div>
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
import { useRelationshipStore } from '@/stores/relationship'
import DynamicForm from '@/components/forms/DynamicForm.vue'

const router = useRouter()
const schemaStore = useSchemaStore()
const ciStore = useCIStore()
const relationshipStore = useRelationshipStore()

// Form state
const loading = ref(false)
const selectedSourceCI = ref(null)
const selectedTargetCI = ref(null)
const selectedRelationshipSchema = ref(null)
const availableCIs = ref([])
const availableTargetCIs = ref([])
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
  return selectedRelationshipSchema.value && schemaAttributes.value.length > 0
})

// Methods
const loadAvailableCIs = async () => {
  try {
    const response = await ciStore.fetchCIs({
      page: 1,
      page_size: 1000,
      status: 'active'
    })
    availableCIs.value = response.data || []
    availableTargetCIs.value = response.data || []
  } catch (error) {
    console.error('Failed to load available CIs:', error)
  }
}

const loadAvailableSchemas = async () => {
  try {
    const response = await schemaStore.fetchRelationshipTypeSchemas({
      page: 1,
      page_size: 100,
      is_active: true
    })
    availableSchemas.value = response.schemas || []
  } catch (error) {
    console.error('Failed to load available schemas:', error)
  }
}

const handleSourceCIChange = (ciId) => {
  if (!ciId) {
    selectedTargetCI.value = null
    return
  }
  
  // Filter out the source CI from target options to prevent self-relationships
  availableTargetCIs.value = availableCIs.value.filter(ci => ci.id !== ciId)
  
  // Clear target CI if it's the same as source
  if (selectedTargetCI.value === ciId) {
    selectedTargetCI.value = null
  }
}

const handleTargetCIChange = (ciId) => {
  // Additional validation can be added here
  console.log('Target CI changed:', ciId)
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
  if (!selectedSourceCI.value || !selectedTargetCI.value || !selectedRelationshipSchema.value) {
    console.warn('Missing required fields for relationship creation')
    return
  }

  try {
    loading.value = true
    
    // Prepare relationship data
    const relationshipData = {
      source_id: selectedSourceCI.value,
      target_id: selectedTargetCI.value,
      schema_id: selectedRelationshipSchema.value,
      schema_name: availableSchemas.value.find(s => s.id === selectedRelationshipSchema.value)?.name,
      ...data
    }
    
    // Create relationship using the relationship store
    await relationshipStore.createRelationship(relationshipData)
    
    // Navigate back to relationships list
    router.push('/relationships')
  } catch (error) {
    console.error('Failed to create relationship:', error)
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

const getSourceCIName = (ciId) => {
  const ci = availableCIs.value.find(c => c.id === ciId)
  return ci ? ci.name : 'Unknown'
}

const getTargetCIName = (ciId) => {
  const ci = availableTargetCIs.value.find(c => c.id === ciId)
  return ci ? ci.name : 'Unknown'
}

const getRelationshipSchemaName = (schemaId) => {
  const schema = availableSchemas.value.find(s => s.id === schemaId)
  return schema ? schema.name : 'Unknown'
}

const goBack = () => {
  router.push('/relationships')
}

// Lifecycle
onMounted(() => {
  loadAvailableCIs()
  loadAvailableSchemas()
})

// Watch for schema changes
watch(selectedRelationshipSchema, (newSchemaId) => {
  if (newSchemaId) {
    handleSchemaChange(newSchemaId)
  }
})
</script>

<style scoped>
.relationship-create-container {
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

.relationship-preview {
  background: #f8f9fa;
  border-radius: 8px;
  padding: 16px;
}

.relationship-preview h4 {
  margin: 0 0 16px 0;
  font-size: 16px;
  color: #606266;
}

.preview-content {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 16px;
  flex-wrap: wrap;
}

.source-chip,
.relationship-chip,
.target-chip {
  font-weight: 500;
}

.arrow-icon {
  color: #909399;
}

.v-card {
  border-radius: 8px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.v-card-text {
  padding: 24px;
}
</style>
