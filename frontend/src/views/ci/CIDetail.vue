<template>
  <div class="ci-detail-container">
    <div class="page-header">
      <h1>Configuration Item Details</h1>
      <div class="header-actions">
        <v-btn @click="goBack" variant="text">
          <v-icon left>mdi-arrow-left</v-icon>
          Back to CIs
        </v-btn>
        <v-btn color="primary" @click="goToEdit(ci.id)">
          <v-icon left>mdi-pencil</v-icon>
          Edit
        </v-btn>
        <v-btn color="error" @click="confirmDelete">
          <v-icon left>mdi-delete</v-icon>
          Delete
        </v-btn>
      </div>
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
        class="mb-4"
      />
      <v-skeleton-loader
        type="card"
        :loading="loading"
      />
    </div>

    <div v-else-if="ci" class="ci-content">
      <!-- Basic Information Card -->
      <v-card class="detail-card">
        <v-card-title class="d-flex justify-space-between align-center">
          <h3>Basic Information</h3>
          <v-chip :color="getStatusColor(ci.status)">
            {{ ci.status }}
          </v-chip>
        </v-card-title>
        <v-card-text>
          <v-row>
            <v-col cols="12" md="6">
              <v-text-field
                :model-value="ci.name"
                label="Name"
                readonly
                prepend-inner-icon="mdi-server"
              />
            </v-col>
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
                :model-value="ci.environment"
                label="Environment"
                readonly
                prepend-inner-icon="mdi-earth"
              >
                <template v-slot:append>
                  <v-chip size="small" :color="getEnvironmentColor(ci.environment)">
                    {{ ci.environment }}
                  </v-chip>
                </template>
              </v-text-field>
            </v-col>
            <v-col cols="12" md="6">
              <v-text-field
                :model-value="ci.owner"
                label="Owner"
                readonly
                prepend-inner-icon="mdi-account"
              />
            </v-col>
            <v-col cols="12" md="6">
              <v-text-field
                :model-value="ci.team"
                label="Team"
                readonly
                prepend-inner-icon="mdi-account-group"
              />
            </v-col>
            <v-col cols="12" md="6">
              <v-text-field
                :model-value="ci.business_unit"
                label="Business Unit"
                readonly
                prepend-inner-icon="mdi-domain"
              />
            </v-col>
            <v-col cols="12" md="6">
              <v-text-field
                :model-value="ci.cost_center"
                label="Cost Center"
                readonly
                prepend-inner-icon="mdi-currency-usd"
              />
            </v-col>
            <v-col cols="12" md="6">
              <v-text-field
                :model-value="formatDate(ci.created_at)"
                label="Created"
                readonly
                prepend-inner-icon="mdi-calendar-plus"
              />
            </v-col>
            <v-col cols="12" md="6">
              <v-text-field
                :model-value="formatDate(ci.updated_at)"
                label="Updated"
                readonly
                prepend-inner-icon="mdi-calendar-edit"
              />
            </v-col>
          </v-row>
          <div v-if="ci.description" class="description-section">
            <h4>Description</h4>
            <p>{{ ci.description }}</p>
          </div>
        </v-card-text>
      </v-card>

      <!-- Schema Attributes Card -->
      <v-card v-if="schemaAttributes.length > 0" class="detail-card">
        <v-card-title>
          <h3>Schema Attributes</h3>
        </v-card-title>
        <v-card-text>
          <v-row>
            <v-col
              v-for="attribute in schemaAttributes"
              :key="attribute.name"
              cols="12"
              md="6"
            >
              <v-text-field
                :model-value="formatAttributeValue(ci[attribute.name], attribute.type)"
                :label="formatAttributeName(attribute.name)"
                readonly
                prepend-inner-icon="getAttributeIcon(attribute.type)"
              >
                <template v-slot:append v-if="attribute.required">
                  <v-icon small color="error">mdi-asterisk</v-icon>
                </template>
              </v-text-field>
            </v-col>
          </v-row>
        </v-card-text>
      </v-card>

      <!-- Technical Details Card -->
      <v-card class="detail-card">
        <v-card-title>
          <h3>Technical Details</h3>
        </v-card-title>
        <v-card-text>
          <v-row>
            <v-col cols="12" md="6">
              <v-text-field
                :model-value="ci.ip_address || 'N/A'"
                label="IP Address"
                readonly
                prepend-inner-icon="mdi-ip-network"
              />
            </v-col>
            <v-col cols="12" md="6">
              <v-text-field
                :model-value="ci.hostname || 'N/A'"
                label="Hostname"
                readonly
                prepend-inner-icon="mdi-laptop"
              />
            </v-col>
            <v-col cols="12" md="6">
              <v-text-field
                :model-value="ci.location || 'N/A'"
                label="Location"
                readonly
                prepend-inner-icon="mdi-map-marker"
              />
            </v-col>
            <v-col cols="12" md="6">
              <v-text-field
                :model-value="ci.datacenter || 'N/A'"
                label="Datacenter"
                readonly
                prepend-inner-icon="mdi-building"
              />
            </v-col>
          </v-row>
          <div v-if="ci.technical_specs" class="specs-section">
            <h4>Technical Specifications</h4>
            <pre>{{ ci.technical_specs }}</pre>
          </div>
        </v-card-text>
      </v-card>

      <!-- Service Information Card -->
      <v-card class="detail-card">
        <v-card-title>
          <h3>Service Information</h3>
        </v-card-title>
        <v-card-text>
          <v-row>
            <v-col cols="12" md="6">
              <v-text-field
                :model-value="ci.service_name || 'N/A'"
                label="Service Name"
                readonly
                prepend-inner-icon="mdi-service-box"
              />
            </v-col>
            <v-col cols="12" md="6">
              <v-text-field
                :model-value="ci.service_level || 'N/A'"
                label="Service Level"
                readonly
                prepend-inner-icon="mdi-star"
              >
                <template v-slot:append v-if="ci.service_level">
                  <v-chip size="small" :color="getServiceLevelColor(ci.service_level)">
                    {{ ci.service_level }}
                  </v-chip>
                </template>
              </v-text-field>
            </v-col>
            <v-col cols="12" md="6">
              <v-text-field
                :model-value="ci.criticality || 'N/A'"
                label="Criticality"
                readonly
                prepend-inner-icon="mdi-alert"
              >
                <template v-slot:append v-if="ci.criticality">
                  <v-chip size="small" :color="getCriticalityColor(ci.criticality)">
                    {{ ci.criticality }}
                  </v-chip>
                </template>
              </v-text-field>
            </v-col>
            <v-col cols="12" md="6">
              <v-text-field
                :model-value="ci.support_hours || 'N/A'"
                label="Support Hours"
                readonly
                prepend-inner-icon="mdi-clock-outline"
              />
            </v-col>
          </v-row>
        </v-card-text>
      </v-card>

      <!-- Relationships Card -->
      <v-card class="detail-card">
        <v-card-title>
          <h3>Relationships</h3>
        </v-card-title>
        <v-card-text>
          <div v-if="relationships.length > 0">
            <v-data-table
              :items="relationships"
              :headers="relationshipHeaders"
              class="elevation-1"
            >
              <template v-slot:item.target_type="{ item }">
                <v-chip size="small" :color="getTypeColor(item.target_type)">
                  {{ item.target_type }}
                </v-chip>
              </template>
              <template v-slot:item.actions="{ item }">
                <v-btn size="small" @click="goToDetail(item.target_id)">
                  <v-icon small>mdi-eye</v-icon>
                  View
                </v-btn>
              </template>
            </v-data-table>
          </div>
          <div v-else class="empty-state">
            <v-icon large color="grey">mdi-link-off</v-icon>
            <p class="text-grey mt-2">No relationships found</p>
          </div>
        </v-card-text>
      </v-card>

      <!-- Audit History Card -->
      <v-card class="detail-card">
        <v-card-title>
          <h3>Audit History</h3>
        </v-card-title>
        <v-card-text>
          <v-timeline>
            <v-timeline-item
              v-for="event in auditHistory"
              :key="event.id"
              :dot-color="getEventColor(event.event_type)"
              size="small"
            >
              <div class="d-flex justify-space-between align-start">
                <div>
                  <h4>{{ event.event_type }}</h4>
                  <p>{{ event.description }}</p>
                  <p v-if="event.user" class="event-user">By: {{ event.user }}</p>
                </div>
                <div class="text-caption text-grey">
                  {{ formatDate(event.created_at) }}
                </div>
              </div>
            </v-timeline-item>
          </v-timeline>
          <div v-if="auditHistory.length === 0" class="empty-state">
            <v-icon large color="grey">mdi-history</v-icon>
            <p class="text-grey mt-2">No audit history available</p>
          </div>
        </v-card-text>
      </v-card>
    </div>

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

    <!-- Delete Confirmation Dialog -->
    <v-dialog v-model="deleteDialog.visible" max-width="400px">
      <v-card>
        <v-card-title>Confirm Delete</v-card-title>
        <v-card-text>
          <p>Are you sure you want to delete the configuration item "{{ ci?.name }}"?</p>
          <p>This action cannot be undone.</p>
        </v-card-text>
        <v-card-actions>
          <v-spacer></v-spacer>
          <v-btn @click="deleteDialog.visible = false">Cancel</v-btn>
          <v-btn
            color="error"
            @click="deleteCI"
            :loading="deleteDialog.loading"
          >
            Delete
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useSchemaStore } from '@/stores/schema'
import { useCIStore } from '@/stores/ci'
import dayjs from 'dayjs'

const route = useRoute()
const router = useRouter()
const schemaStore = useSchemaStore()
const ciStore = useCIStore()

// State
const loading = ref(false)
const ci = ref(null)
const schemaAttributes = ref([])
const relationships = ref([])
const auditHistory = ref([])
const deleteDialog = reactive({
  visible: false,
  loading: false
})

// Table headers
const relationshipHeaders = [
  { text: 'Related CI', value: 'target_name' },
  { text: 'Relationship Type', value: 'relationship_type' },
  { text: 'Type', value: 'target_type' },
  { text: 'Actions', value: 'actions', sortable: false, align: 'end' }
]

// Methods
const fetchCIDetail = async () => {
  loading.value = true
  try {
    const ciData = await ciStore.fetchCIDetail(route.params.id)
    ci.value = ciData
    
    // Load schema attributes if schema_id is available
    if (ciData.schema_id) {
      try {
        const schemaResponse = await schemaStore.fetchCiTypeSchemaDetail(ciData.schema_id)
        schemaAttributes.value = schemaResponse.attributes || []
      } catch (error) {
        console.error('Failed to load schema details:', error)
        schemaAttributes.value = []
      }
    } else {
      schemaAttributes.value = []
    }
  } catch (error) {
    console.error('Failed to fetch CI details:', error)
    ci.value = null
  } finally {
    loading.value = false
  }
}

const fetchRelationships = async () => {
  try {
    const response = await ciStore.fetchRelationships({ source_id: route.params.id })
    relationships.value = response.data || []
  } catch (error) {
    console.error('Failed to fetch relationships:', error)
  }
}

const fetchAuditHistory = async () => {
  try {
    const response = await ciStore.fetchAuditHistory(route.params.id)
    auditHistory.value = response.data || []
  } catch (error) {
    console.error('Failed to fetch audit history:', error)
  }
}

const goBack = () => {
  router.push('/cis')
}

const goToEdit = (id) => {
  router.push(`/cis/${id}/edit`)
}

const goToDetail = (id) => {
  router.push(`/cis/${id}`)
}

const confirmDelete = () => {
  deleteDialog.visible = true
}

const deleteCI = async () => {
  if (!ci.value) return
  
  deleteDialog.loading = true
  try {
    await ciStore.deleteCI(ci.value.id)
    deleteDialog.visible = false
    router.push('/cis')
  } catch (error) {
    console.error('Failed to delete CI:', error)
  } finally {
    deleteDialog.loading = false
  }
}

const getTypeColor = (type) => {
  const types = {
    server: 'primary',
    database: 'success',
    application: 'warning',
    network: 'info',
    storage: 'error'
  }
  return types[type] || 'grey'
}

const getStatusColor = (status) => {
  const statuses = {
    active: 'success',
    inactive: 'info',
    maintenance: 'warning',
    decommissioned: 'error'
  }
  return statuses[status] || 'grey'
}

const getEnvironmentColor = (environment) => {
  const environments = {
    development: 'primary',
    staging: 'warning',
    production: 'error',
    testing: 'info'
  }
  return environments[environment] || 'grey'
}

const getServiceLevelColor = (level) => {
  const levels = {
    gold: 'success',
    silver: 'warning',
    bronze: 'info'
  }
  return levels[level] || 'grey'
}

const getCriticalityColor = (criticality) => {
  const criticalities = {
    high: 'error',
    medium: 'warning',
    low: 'success'
  }
  return criticalities[criticality] || 'grey'
}

const getEventColor = (eventType) => {
  const types = {
    created: 'success',
    updated: 'warning',
    deleted: 'error',
    restored: 'info'
  }
  return types[eventType] || 'primary'
}

const getAttributeIcon = (type) => {
  const icons = {
    string: 'mdi-text',
    number: 'mdi-numeric',
    boolean: 'mdi-toggle-switch',
    date: 'mdi-calendar',
    array: 'mdi-array',
    object: 'mdi-code-braces'
  }
  return icons[type] || 'mdi-text'
}

const formatAttributeName = (name) => {
  return name.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase())
}

const formatAttributeValue = (value, type) => {
  if (value === undefined || value === null || value === '') {
    return 'N/A'
  }
  
  switch (type) {
    case 'boolean':
      return value ? 'Yes' : 'No'
    case 'array':
      return Array.isArray(value) ? value.join(', ') : String(value)
    case 'object':
      return typeof value === 'object' ? JSON.stringify(value) : String(value)
    case 'date':
      return formatDate(value)
    default:
      return String(value)
  }
}

const formatDate = (date) => {
  return dayjs(date).format('YYYY-MM-DD HH:mm:ss')
}

// Lifecycle
onMounted(async () => {
  await fetchCIDetail()
  await fetchRelationships()
  await fetchAuditHistory()
})
</script>

<style scoped>
.ci-detail-container {
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

.header-actions {
  display: flex;
  gap: 10px;
}

.loading-container {
  padding: 20px;
}

.ci-content {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.detail-card {
  margin-bottom: 20px;
}

.description-section,
.specs-section {
  margin-top: 20px;
}

.description-section h4,
.specs-section h4 {
  margin: 0 0 10px 0;
  font-size: 16px;
  color: #606266;
}

.description-section p {
  margin: 0;
  line-height: 1.6;
  color: #606266;
}

.specs-section pre {
  background: #f5f5f5;
  padding: 15px;
  border-radius: 4px;
  white-space: pre-wrap;
  word-wrap: break-word;
  margin: 0;
}

.empty-state {
  padding: 40px 0;
  text-align: center;
}

.event-user {
  margin: 5px 0 0 0;
  font-size: 12px;
  color: #909399;
  font-style: italic;
}

.error-state {
  padding: 40px 0;
  text-align: center;
}

.v-chip {
  text-transform: capitalize;
}

.v-skeleton-loader {
  border-radius: 8px;
  margin-bottom: 16px;
}
</style>
