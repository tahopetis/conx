<template>
  <div class="relationships-container">
    <div class="page-header">
      <h1>Relationships</h1>
      <v-btn color="primary" @click="goToCreate">
        <v-icon left>mdi-plus</v-icon>
        Create Relationship
      </v-btn>
    </div>

    <!-- Filters -->
    <div class="filters-section">
      <v-card>
        <v-card-text>
          <v-row>
            <v-col cols="12" md="3">
              <v-text-field
                v-model="filters.search"
                label="Search Relationships..."
                clearable
                @click:clear="handleSearch"
                @keyup.enter="handleSearch"
                prepend-inner-icon="mdi-magnify"
                variant="outlined"
                density="compact"
              />
            </v-col>
            <v-col cols="12" md="2">
              <v-select
                v-model="filters.schema_id"
                label="Schema"
                clearable
                @update:model-value="handleSearch"
                :items="schemaOptions"
                item-title="name"
                item-value="id"
                variant="outlined"
                density="compact"
              />
            </v-col>
            <v-col cols="12" md="2">
              <v-select
                v-model="filters.source_ci_id"
                label="Source CI"
                clearable
                @update:model-value="handleSearch"
                :items="ciOptions"
                item-title="name"
                item-value="id"
                variant="outlined"
                density="compact"
              />
            </v-col>
            <v-col cols="12" md="2">
              <v-select
                v-model="filters.target_ci_id"
                label="Target CI"
                clearable
                @update:model-value="handleSearch"
                :items="ciOptions"
                item-title="name"
                item-value="id"
                variant="outlined"
                density="compact"
              />
            </v-col>
            <v-col cols="12" md="2">
              <v-select
                v-model="filters.direction"
                label="Direction"
                clearable
                @update:model-value="handleSearch"
                :items="directionOptions"
                variant="outlined"
                density="compact"
              />
            </v-col>
            <v-col cols="12" md="1">
              <v-btn @click="resetFilters" variant="outlined" block>
                <v-icon>mdi-refresh</v-icon>
                Reset
              </v-btn>
            </v-col>
          </v-row>
        </v-card-text>
      </v-card>
    </div>

    <!-- Relationship Table -->
    <v-card class="table-card">
      <v-card-text>
        <v-data-table
          v-model:items-per-page="pagination.limit"
          :headers="tableHeaders"
          :items="relationships"
          :loading="loading"
          :items-length="pagination.total"
          :page="pagination.page"
          @update:options="handleTableUpdate"
          class="elevation-1"
        >
          <template v-slot:item.source_ci_name="{ item }">
            <v-btn
              variant="text"
              color="primary"
              @click="goToCIDetail(item.source_id)"
              class="text-left"
            >
              {{ item.source_ci_name }}
            </v-btn>
          </template>
          
          <template v-slot:item.target_ci_name="{ item }">
            <v-btn
              variant="text"
              color="primary"
              @click="goToCIDetail(item.target_id)"
              class="text-left"
            >
              {{ item.target_ci_name }}
            </v-btn>
          </template>
          
          <template v-slot:item.schema_name="{ item }">
            <v-chip size="small" :color="getRelationshipColor(item.schema_name)">
              {{ item.schema_name }}
            </v-chip>
          </template>
          
          <template v-slot:item.direction="{ item }">
            <v-icon size="small" :color="getDirectionColor(item.direction)">
              {{ getDirectionIcon(item.direction) }}
            </v-icon>
            <span class="ml-1">{{ formatDirection(item.direction) }}</span>
          </template>
          
          <template v-slot:item.created_at="{ item }">
            {{ formatDate(item.created_at) }}
          </template>
          
          <template v-slot:item.updated_at="{ item }">
            {{ formatDate(item.updated_at) }}
          </template>
          
          <template v-slot:item.actions="{ item }">
            <div class="action-buttons">
              <v-btn size="small" icon @click="goToDetail(item.id)" color="primary">
                <v-icon small>mdi-eye</v-icon>
              </v-btn>
              <v-btn size="small" icon @click="goToEdit(item.id)" color="warning">
                <v-icon small>mdi-pencil</v-icon>
              </v-btn>
              <v-btn size="small" icon @click="confirmDelete(item)" color="error">
                <v-icon small>mdi-delete</v-icon>
              </v-btn>
            </div>
          </template>
        </v-data-table>
      </v-card-text>
    </v-card>

    <!-- Delete Confirmation Dialog -->
    <v-dialog v-model="deleteDialog.visible" max-width="400px">
      <v-card>
        <v-card-title>Confirm Delete</v-card-title>
        <v-card-text>
          <p>Are you sure you want to delete the relationship "{{ deleteDialog.relationship?.schema_name }}"?</p>
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
import { ref, reactive, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useRelationshipStore } from '@/stores/relationship'
import { useSchemaStore } from '@/stores/schema'
import { useCIStore } from '@/stores/ci'
import dayjs from 'dayjs'

const router = useRouter()
const relationshipStore = useRelationshipStore()
const schemaStore = useSchemaStore()
const ciStore = useCIStore()

// State
const loading = ref(false)
const relationships = ref([])
const availableSchemas = ref([])
const availableCIs = ref([])
const filters = reactive({
  search: '',
  schema_id: '',
  source_ci_id: '',
  target_ci_id: '',
  direction: ''
})
const pagination = reactive({
  page: 1,
  limit: 20,
  total: 0,
  sort: 'created_at',
  order: 'desc'
})
const deleteDialog = reactive({
  visible: false,
  relationship: null,
  loading: false
})

// Options for select fields
const directionOptions = [
  { title: 'Source → Target', value: 'forward' },
  { title: 'Target → Source', value: 'backward' },
  { title: 'Bidirectional', value: 'bidirectional' }
]

const schemaOptions = computed(() => {
  return availableSchemas.value.map(schema => ({
    title: schema.name,
    value: schema.id
  }))
})

const ciOptions = computed(() => {
  return availableCIs.value.map(ci => ({
    title: ci.name,
    value: ci.id
  }))
})

// Table headers
const tableHeaders = [
  { title: 'Source CI', key: 'source_ci_name', sortable: true, width: '200px' },
  { title: 'Target CI', key: 'target_ci_name', sortable: true, width: '200px' },
  { title: 'Relationship Type', key: 'schema_name', sortable: true, width: '150px' },
  { title: 'Direction', key: 'direction', sortable: true, width: '120px' },
  { title: 'Created', key: 'created_at', sortable: true, width: '180px' },
  { title: 'Updated', key: 'updated_at', sortable: true, width: '180px' },
  { title: 'Actions', key: 'actions', sortable: false, width: '120px', align: 'end' }
]

// Methods
const fetchRelationships = async () => {
  loading.value = true
  try {
    const params = {
      page: pagination.page,
      limit: pagination.limit,
      sort: pagination.sort,
      order: pagination.order,
      ...Object.fromEntries(Object.entries(filters).filter(([_, v]) => v !== ''))
    }
    
    const response = await relationshipStore.fetchRelationships(params)
    relationships.value = response.data || []
    pagination.total = response.total || 0
  } catch (error) {
    console.error('Failed to fetch relationships:', error)
  } finally {
    loading.value = false
  }
}

const fetchAvailableSchemas = async () => {
  try {
    const response = await schemaStore.fetchRelationshipTypeSchemas({
      page: 1,
      page_size: 100,
      is_active: true
    })
    availableSchemas.value = response.schemas || []
  } catch (error) {
    console.error('Failed to fetch available schemas:', error)
  }
}

const fetchAvailableCIs = async () => {
  try {
    const response = await ciStore.fetchCIs({
      page: 1,
      page_size: 1000,
      status: 'active'
    })
    availableCIs.value = response.data || []
  } catch (error) {
    console.error('Failed to fetch available CIs:', error)
  }
}

const handleSearch = () => {
  pagination.page = 1
  fetchRelationships()
}

const resetFilters = () => {
  filters.search = ''
  filters.schema_id = ''
  filters.source_ci_id = ''
  filters.target_ci_id = ''
  filters.direction = ''
  pagination.page = 1
  fetchRelationships()
}

const handleTableUpdate = (options) => {
  pagination.page = options.page
  pagination.limit = options.itemsPerPage
  pagination.sort = options.sortBy[0]?.key || 'created_at'
  pagination.order = options.sortBy[0]?.order === 'asc' ? 'asc' : 'desc'
  fetchRelationships()
}

const goToCreate = () => {
  router.push('/relationships/create')
}

const goToDetail = (id) => {
  router.push(`/relationships/${id}`)
}

const goToEdit = (id) => {
  router.push(`/relationships/${id}/edit`)
}

const goToCIDetail = (id) => {
  router.push(`/cis/${id}`)
}

const confirmDelete = (relationship) => {
  deleteDialog.relationship = relationship
  deleteDialog.visible = true
}

const deleteRelationship = async () => {
  if (!deleteDialog.relationship) return
  
  deleteDialog.loading = true
  try {
    await relationshipStore.deleteRelationship(deleteDialog.relationship.id)
    deleteDialog.visible = false
    fetchRelationships()
  } catch (error) {
    console.error('Failed to delete relationship:', error)
  } finally {
    deleteDialog.loading = false
  }
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

const getDirectionColor = (direction) => {
  const colors = {
    forward: 'primary',
    backward: 'warning',
    bidirectional: 'success'
  }
  return colors[direction] || 'grey'
}

const getDirectionIcon = (direction) => {
  const icons = {
    forward: 'mdi-arrow-right',
    backward: 'mdi-arrow-left',
    bidirectional: 'mdi-arrow-left-right'
  }
  return icons[direction] || 'mdi-arrow-right'
}

const formatDirection = (direction) => {
  const formats = {
    forward: 'Forward',
    backward: 'Backward',
    bidirectional: 'Bidirectional'
  }
  return formats[direction] || direction
}

const formatDate = (date) => {
  return dayjs(date).format('YYYY-MM-DD HH:mm:ss')
}

// Lifecycle
onMounted(async () => {
  await fetchAvailableSchemas()
  await fetchAvailableCIs()
  await fetchRelationships()
})
</script>

<style scoped>
.relationships-container {
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

.filters-section {
  margin-bottom: 20px;
}

.table-card {
  margin-bottom: 20px;
}

.action-buttons {
  display: flex;
  gap: 8px;
}

.v-chip {
  text-transform: capitalize;
}

.v-data-table {
  margin-top: 10px;
}

.v-btn.text-left {
  justify-content: flex-start;
  text-align: left;
}
</style>
