<template>
  <div class="relationship-edit-container">
    <div class="page-header">
      <h1>Edit Relationship</h1>
      <v-btn @click="goBack" variant="text">
        <v-icon left>mdi-arrow-left</v-icon>
        Back to Relationship Details
      </v-btn>
    </div>

    <div v-if="loading" class="loading-container">
      <v-skeleton-loader
        type="card"
        :loading="loading"
        class="mb-4"
      />
      <v-skeleton-loader
        type="card"
        :loading="loading"
        class="mb-4"
      />
      <v-skeleton-loader
        type="card"
        :loading="loading"
      />
    </div>

    <v-card v-else-if="relationship" class="form-card">
      <v-card-text>
        <!-- Basic Relationship Information (Read-only) -->
        <v-row class="mb-6">
          <v-col cols="12" md="6">
            <v-text-field
              :model-value="getSourceCIName(relationship.source_id)"
              label="Source Configuration Item"
              readonly
              prepend-inner-icon="mdi-server"
            />
          </v-col>
          <v-col cols="12" md="6">
            <v-text-field
              :model-value="getTargetCIName(relationship.target_id)"
              label="Target Configuration Item"
              readonly
              prepend-inner-icon="mdi-server-network"
            />
          </v-col>
        </v-row>

        <v-row class="mb-6">
          <v-col cols="12">
            <v-text-field
              :model-value="relationship.schema_name"
              label="Relationship Type Schema"
              readonly
              prepend-inner-icon="mdi-link-variant"
            />
          </v-col>
        </v-row>

        <!-- Dynamic Form for Relationship Attributes -->
        <div v-if="schemaAttributes.length > 0">
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
                  :disabled="!isFormValid"
                >
                  <v-icon left>mdi-check</v-icon>
                  Update Relationship
                </v-btn>
              </div>
            </template>
          </DynamicForm>
        </div>

        <!-- Empty State -->
        <div v-else class="text-center py-8">
          <v-icon large color="grey">mdi-alert-circle-outline</v-icon>
          <p class="text-grey mt-2">No attributes found for this relationship's schema. Please add attributes to the schema or contact your administrator.</p>
        </div>

        <!-- Relationship Preview -->
        <div class="relationship-preview mt-6">
          <v-divider class="mb-4"></v-divider>
          <h4>Relationship Preview</h4>
          <div class="preview-content">
            <v-chip color="primary" class="source-chip">
              {{ getSourceCIName(relationship.source_id) }}
            </v-chip>
            <v-icon class="arrow-icon">mdi-arrow-right</v-icon>
            <v-chip color="secondary" class="relationship-chip">
              {{ relationship.schema_name }}
            </v-chip>
            <v-icon class="arrow-icon">mdi-arrow-right</v-icon>
            <v-chip color="success" class="target-chip">
              {{ getTargetCIName(relationship.target_id) }}
            </v-chip>
          </div>
        </div>
      </v-card-text>
    </v-card>

    <div v-else class="error-state">
      <v-card>
        <v-card-text class="text-center">
          <v-icon size="64" color="error">mdi-alert-circle</v-icon>
          <h2 class="text-h5 mt-4">Relationship Not Found</h2>
          <p class="text-body-1 mt-2">The requested relationship could not be found.</p>
          <v-btn color="primary" class="mt-4" @click="goBack">
            <v-icon left>mdi-arrow-left</v-icon>
            Back to Relationships
          </v-btn>
        </v-card-text>
      </v-card>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useSchemaStore } from '@/stores/schema'
import { useCIStore } from '@/stores/ci'
import { useRelationshipStore } from '@/stores/relationship'
import DynamicForm from '@/components/forms/DynamicForm.vue'

const route = useRoute()
const router = useRouter()
const schemaStore = useSchemaStore()
const ciStore = useCIStore()
const relationshipStore = useRelationshipStore()

// Form state
const loading = ref(false)
const relationship = ref(null)
const availableCIs = ref([])
const schemaAttributes = ref([])
const isFormValid = ref(false)

// Form data
const formData = reactive({})

// Computed properties
const isFormReady = computed(() => {
  return relationship.value && schemaAttributes.value.length > 0
})

// Methods
const fetchRelationshipDetail = async () => {
  loading.value = true
  try {
    // Fetch relationship details
    const relationshipData = await relationshipStore.fetchRelationshipDetail(route.params.id)
    relationship.value = relationshipData
    
    // Load available CIs for display
    await loadAvailableCIs()
    
    // Load schema attributes if schema_id is available
    if (relationshipData.schema_id) {
      try {
        const schemaResponse = await schemaStore.fetchRelationshipTypeSchemaDetail(relationshipData.schema_id)
        schemaAttributes.value = schemaResponse.attributes || []
        
        // Initialize form data with relationship data
        Object.keys(formData).forEach(key => {
          delete formData[key]
        })
        
        // Populate form with relationship data, matching schema attributes
        schemaAttributes.value.forEach(attr => {
          if (relationshipData[attr.name] !== undefined) {
            formData[attr.name] = relationshipData[attr.name]
          } else if (attr.default !== undefined && attr.default !== '') {
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
      } catch (error) {
        console.error('Failed to load schema details:', error)
        schemaAttributes.value = []
      }
    } else {
      // Fallback to basic attributes if no schema
      schemaAttributes.value = []
    }
  } catch (error) {
    console.error('Failed to fetch relationship details:', error)
    relationship.value = null
  } finally {
    loading.value = false
  }
}

const loadAvailableCIs = async () => {
  try {
    const response = await ciStore.fetchCIs({
      page: 1,
      page_size: 1000,
      status: 'active'
    })
    availableCIs.value = response.data || []
  } catch (error) {
    console.error('Failed to load available CIs:', error)
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
  if (!relationship.value) {
    console.warn('No relationship data available')
    return
  }

  try {
    loading.value = true
    
    // Prepare relationship data for update
    const relationshipData = {
      ...data,
      schema_id: relationship.value.schema_id,
      schema_name: relationship.value.schema_name
    }
    
    // Update relationship using the relationship store
    await relationshipStore.updateRelationship({
      id: route.params.id,
      data: relationshipData
    })
    
    // Navigate back to relationship details
    router.push(`/relationships/${route.params.id}`)
  } catch (error) {
    console.error('Failed to update relationship:', error)
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
  const ci = availableCIs.value.find(c => c.id === ciId)
  return ci ? ci.name : 'Unknown'
}

const goBack = () => {
  router.push(`/relationships/${route.params.id}`)
}

// Lifecycle
onMounted(() => {
  fetchRelationshipDetail()
})
</script>

<style scoped>
.relationship-edit-container {
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

.loading-container {
  padding: 20px;
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

.error-state {
  padding: 40px 0;
  text-align: center;
}

.v-skeleton-loader {
  border-radius: 8px;
  margin-bottom: 16px;
}
</style>
