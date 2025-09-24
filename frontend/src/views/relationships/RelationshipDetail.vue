<template>
  <div class="relationship-detail-container">
    <div class="page-header">
      <h1>Relationship Details</h1>
      <div class="header-actions">
        <v-btn @click="goBack" variant="text">
          <v-icon left>mdi-arrow-left</v-icon>
          Back to Relationships
        </v-btn>
        <v-btn color="primary" @click="goToEdit(relationship.id)">
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
      />
    </div>

    <div v-else-if="relationship" class="relationship-content">
      <!-- Basic Information Card -->
      <v-card class="detail-card">
        <v-card-title class="d-flex justify-space-between align-center">
          <h3>Basic Information</h3>
          <v-chip :color="getRelationshipColor(relationship.schema_name)">
            {{ relationship.schema_name }}
          </v-chip>
        </v-card-title>
        <v-card-text>
          <v-row>
            <v-col cols="12" md="6">
              <v-text-field
                :model-value="getSourceCIName(relationship.source_id)"
                label="Source Configuration Item"
                readonly
                prepend-inner-icon="mdi-server"
              >
                <template v-slot:append>
                  <v-btn size="small" icon @click="goToCIDetail(relationship.source_id)" color="primary">
                    <v-icon small>mdi-open-in-new</v-icon>
                  </v-btn>
                </template>
              </v-text-field>
            </v-col>
            <v-col cols="12" md="6">
              <v-text-field
                :model-value="getTargetCIName(relationship.target_id)"
                label="Target Configuration Item"
                readonly
                prepend-inner-icon="mdi-server-network"
              >
                <template v-slot:append>
                  <v-btn size="small" icon @click="goToCIDetail(relationship.target_id)" color="primary">
                    <v-icon small>mdi-open-in-new</v-icon>
                  </v-btn>
                </template>
              </v-text-field>
            </v-col>
            <v-col cols="12" md="6">
              <v-text-field
                :model-value="relationship.schema_name"
                label="Relationship Type Schema"
                readonly
                prepend-inner-icon="mdi-link-variant"
              />
            </v-col>
            <v-col cols="12" md="6">
              <v-text-field
                :model-value="formatDate(relationship.created_at)"
                label="Created"
                readonly
                prepend-inner-icon="mdi-calendar-plus"
              />
            </v-col>
            <v-col cols="12" md="6">
              <v-text-field
                :model-value="formatDate(relationship.updated_at)"
                label="Updated"
                readonly
                prepend-inner-icon="mdi-calendar-edit"
              />
            </v-col>
          </v-row>
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
                :model-value="formatAttributeValue(relationship[attribute.name], attribute.type)"
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

      <!-- Relationship Visualization Card -->
      <v-card class="detail-card">
        <v-card-title>
          <h3>Relationship Visualization</h3>
        </v-card-title>
        <v-card-text>
          <div class="relationship-visualization">
            <div class="visualization-content">
              <div class="ci-node source-node">
                <v-avatar size="64" color="primary">
                  <v-icon size="32">mdi-server</v-icon>
                </v-avatar>
                <div class="ci-info">
                  <h4>{{ getSourceCIName(relationship.source_id) }}</h4>
                  <p>Source CI</p>
                </div>
              </div>
              
              <div class="relationship-arrow">
                <v-icon size="48" color="secondary">mdi-arrow-right</v-icon>
                <div class="relationship-label">
                  <v-chip color="secondary" class="relationship-chip">
                    {{ relationship.schema_name }}
                  </v-chip>
                </div>
              </div>
              
              <div class="ci-node target-node">
                <v-avatar size="64" color="success">
                  <v-icon size="32">mdi-server-network</v-icon>
                </v-avatar>
                <div class="ci-info">
                  <h4>{{ getTargetCIName(relationship.target_id) }}</h4>
                  <p>Target CI</p>
                </div>
              </div>
            </div>
          </div>
        </v-card-text>
      </v-card>

      <!-- Related Relationships Card -->
      <v-card class="detail-card">
        <v-card-title>
          <h3>Related Relationships</h3>
        </v-card-title>
        <v-card-text>
          <div v-if="relatedRelationships.length > 0">
            <v-data-table
              :items="relatedRelationships"
              :headers="relatedRelationshipHeaders"
              class="elevation-1"
            >
              <template v-slot:item.schema_name="{ item }">
                <v-chip size="small" :color="getRelationshipColor(item.schema_name)">
                  {{ item.schema_name }}
                </v-chip>
              </template>
              <template v-slot:item.actions="{ item }">
                <v-btn size="small" @click="goToRelationshipDetail(item.id)">
                  <v-icon small>mdi-eye</v-icon>
                  View
                </v-btn>
              </template>
            </v-data-table>
          </div>
          <div v-else class="empty-state">
            <v-icon large color="grey">mdi-link-off</v-icon>
            <p class="text-grey mt-2">No related relationships found</p>
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
          <h2 class="text-h5 mt-4">Relationship Not Found</h2>
          <p class="text-body-1 mt-2">The requested relationship could not be found.</p>
          <v-btn color="primary" class="mt-4" @click="goBack">
            <v-icon left>mdi-arrow-left</v-icon>
            Back to Relationships
          </v-btn>
        </v-card-text>
      </v-card>
    </div>

    <!-- Delete Confirmation Dialog -->
    <v-dialog v-model="deleteDialog.visible" max-width="400px">
      <v-card>
        <v-card-title>Confirm Delete</v-card-title>
        <v-card-text>
          <p>Are you sure you want to delete the relationship "{{ relationship?.schema_name }}"?</p>
          <p>This action cannot be undone.</p>
        </v-card-text>
        <v-card-actions>
          <v-spacer></v-spacer>
          <v-btn @click="deleteDialog.visible = false">Cancel</v-btn>
          <v-btn
            color="error"
            @click="deleteRelationship"
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
import { useRelationshipStore } from '@/stores/relationship'
import dayjs from 'dayjs'

const route = useRoute()
const router = useRouter()
const schemaStore = useSchemaStore()
const ciStore = useCIStore()
const relationshipStore = useRelationshipStore()

// State
const loading = ref(false)
const relationship = ref(null)
const schemaAttributes = ref([])
const availableCIs = ref([])
const relatedRelationships = ref([])
const auditHistory = ref([])
const deleteDialog = reactive({
  visible: false,
  loading: false
})

// Table headers
const relatedRelationshipHeaders = [
  { text: 'Related CI', value: 'related_ci_name' },
  { text: 'Relationship Type', value: 'schema_name' },
  { text: 'Direction', value: 'direction' },
  { text: 'Actions', value: 'actions', sortable: false, align: 'end' }
]

// Methods
const fetchRelationshipDetail = async () => {
  loading.value = true
  try {
    const relationshipData = await relationshipStore.fetchRelationshipDetail(route.params.id)
    relationship.value = relationshipData
    
    // Load available CIs for display
    await loadAvailableCIs()
    
    // Load schema attributes if schema_id is available
    if (relationshipData.schema_id) {
      try {
        const schemaResponse = await schemaStore.fetchRelationshipTypeSchemaDetail(relationshipData.schema_id)
        schemaAttributes.value = schemaResponse.attributes || []
      } catch (error) {
        console.error('Failed to load schema details:', error)
        schemaAttributes.value = []
      }
    } else {
      schemaAttributes.value = []
    }
    
    // Load related relationships
    await fetchRelatedRelationships()
    
    // Load audit history
    await fetchAuditHistory()
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

const fetchRelatedRelationships = async () => {
  if (!relationship.value) return
  
  try {
    const response = await relationshipStore.fetchRelatedRelationships({
      source_id: relationship.value.source_id,
      target_id: relationship.value.target_id,
      exclude_id: relationship.value.id
    })
    relatedRelationships.value = response.data || []
  } catch (error) {
    console.error('Failed to fetch related relationships:', error)
  }
}

const fetchAuditHistory = async () => {
  try {
    const response = await relationshipStore.fetchAuditHistory(route.params.id)
    auditHistory.value = response.data || []
  } catch (error) {
    console.error('Failed to fetch audit history:', error)
  }
}

const goBack = () => {
  router.push('/relationships')
}

const goToEdit = (id) => {
  router.push(`/relationships/${id}/edit`)
}

const goToCIDetail = (ciId) => {
  router.push(`/cis/${ciId}`)
}

const goToRelationshipDetail = (id) => {
  router.push(`/relationships/${id}`)
}

const confirmDelete = () => {
  deleteDialog.visible = true
}

const deleteRelationship = async () => {
  if (!relationship.value) return
  
  deleteDialog.loading = true
  try {
    await relationshipStore.deleteRelationship(relationship.value.id)
    deleteDialog.visible = false
    router.push('/relationships')
  } catch (error) {
    console.error('Failed to delete relationship:', error)
  } finally {
    deleteDialog.loading = false
  }
}

const getSourceCIName = (ciId) => {
  const ci = availableCIs.value.find(c => c.id === ciId)
  return ci ? ci.name : 'Unknown'
}

const getTargetCIName = (ciId) => {
  const ci = availableCIs.value.find(c => c.id === ciId)
  return ci ? ci.name : 'Unknown'
}

const getRelationshipColor = (schemaName) => {
  const colors = {
    'depends_on': 'primary',
    'connected_to': 'success',
    'hosted_on': 'warning',
    'part_of': 'error',
    'related_to': 'info'
  }
  return colors[schemaName] || 'secondary'
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

const getEventColor = (eventType) => {
  const types = {
    created: 'success',
    updated: 'warning',
    deleted: 'error',
    restored: 'info'
  }
  return types[eventType] || 'primary'
}

const formatDate = (date) => {
  return dayjs(date).format('YYYY-MM-DD HH:mm:ss')
}

// Lifecycle
onMounted(async () => {
  await fetchRelationshipDetail()
})
</script>

<style scoped>
.relationship-detail-container {
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

.relationship-content {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.detail-card {
  margin-bottom: 20px;
}

.relationship-visualization {
  padding: 20px 0;
}

.visualization-content {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 40px;
  flex-wrap: wrap;
}

.ci-node {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  gap: 16px;
}

.ci-info h4 {
  margin: 0 0 4px 0;
  font-size: 16px;
  color: #303133;
}

.ci-info p {
  margin: 0;
  font-size: 14px;
  color: #909399;
}

.relationship-arrow {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
}

.relationship-label {
  text-align: center;
}

.relationship-chip {
  font-weight: 500;
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

.source-node .v-avatar {
  background: linear-gradient(135deg, #1976d2, #42a5f5);
}

.target-node .v-avatar {
  background: linear-gradient(135deg, #388e3c, #66bb6a);
}
</style>
