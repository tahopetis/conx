<template>
  <div class="ci-edit-container">
    <div class="page-header">
      <h1>Edit Configuration Item</h1>
      <v-btn @click="goBack" variant="text">
        <v-icon left>mdi-arrow-left</v-icon>
        Back to CI Details
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

    <v-card v-else-if="ci" class="form-card">
      <v-card-text>
        <!-- Schema Information (Read-only) -->
        <v-row class="mb-6">
          <v-col cols="12" md="6">
            <v-text-field
              :model-value="ci.schema_name"
              label="CI Type Schema"
              readonly
              prepend-inner-icon="mdi-database-outline"
            />
          </v-col>
          <v-col cols="12" md="6">
            <v-text-field
              :model-value="ci.name"
              label="CI Name"
              readonly
              prepend-inner-icon="mdi-server"
            />
          </v-col>
        </v-row>

        <!-- Dynamic Form -->
        <div v-if="schemaAttributes.length > 0">
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
                  Update CI
                </v-btn>
              </div>
            </template>
          </DynamicForm>
        </div>

        <!-- Empty State -->
        <div v-else class="text-center py-8">
          <v-icon large color="grey">mdi-alert-circle-outline</v-icon>
          <p class="text-grey mt-2">No attributes found for this CI's schema. Please add attributes to the schema or contact your administrator.</p>
        </div>
      </v-card-text>
    </v-card>

    <div v-else class="error-state">
      <v-card>
        <v-card-text class="text-center">
          <v-icon size="64" color="error">mdi-alert-circle</v-icon>
          <h2 class="text-h5 mt-4">Configuration Item Not Found</h2>
          <p class="text-body-1 mt-2">The requested configuration item could not be found.</p>
          <v-btn color="primary" class="mt-4" @click="goBack">
            <v-icon left>mdi-arrow-left</v-icon>
            Back to CIs
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
import DynamicForm from '@/components/forms/DynamicForm.vue'

const route = useRoute()
const router = useRouter()
const schemaStore = useSchemaStore()
const ciStore = useCIStore()

// Form state
const formRef = ref()
const loading = ref(false)
const ci = ref(null)
const schemaAttributes = ref([])
const isFormValid = ref(false)

// Form data
const formData = reactive({})

// Computed properties
const isFormReady = computed(() => {
  return ci.value && schemaAttributes.value.length > 0
})

// Methods
const fetchCIDetail = async () => {
  loading.value = true
  try {
    // Fetch CI details
    const ciData = await ciStore.fetchCIDetail(route.params.id)
    ci.value = ciData
    
    // Load schema attributes if schema_id is available
    if (ciData.schema_id) {
      try {
        const schemaResponse = await schemaStore.fetchCiTypeSchemaDetail(ciData.schema_id)
        schemaAttributes.value = schemaResponse.attributes || []
        
        // Initialize form data with CI data
        Object.keys(formData).forEach(key => {
          delete formData[key]
        })
        
        // Populate form with CI data, matching schema attributes
        schemaAttributes.value.forEach(attr => {
          if (ciData[attr.name] !== undefined) {
            formData[attr.name] = ciData[attr.name]
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
    console.error('Failed to fetch CI details:', error)
    ci.value = null
  } finally {
    loading.value = false
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
  if (!ci.value) {
    console.warn('No CI data available')
    return
  }

  try {
    loading.value = true
    
    // Prepare CI data for update
    const ciData = {
      ...data,
      schema_id: ci.value.schema_id,
      schema_name: ci.value.schema_name
    }
    
    // Update CI using the CI store
    await ciStore.updateCI({
      id: route.params.id,
      data: ciData
    })
    
    // Navigate back to CI details
    router.push(`/cis/${route.params.id}`)
  } catch (error) {
    console.error('Failed to update CI:', error)
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
  router.push(`/cis/${route.params.id}`)
}

// Lifecycle
onMounted(() => {
  fetchCIDetail()
})
</script>

<style scoped>
.ci-edit-container {
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
