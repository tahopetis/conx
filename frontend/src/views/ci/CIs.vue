<template>
  <div class="cis-container">
    <div class="page-header">
      <h1>Configuration Items</h1>
      <v-btn color="primary" @click="goToCreate">
        <v-icon left>mdi-plus</v-icon>
        Create CI
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
                label="Search CIs..."
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
                v-model="filters.type"
                label="Type"
                clearable
                @update:model-value="handleSearch"
                :items="typeOptions"
                variant="outlined"
                density="compact"
              />
            </v-col>
            <v-col cols="12" md="2">
              <v-select
                v-model="filters.status"
                label="Status"
                clearable
                @update:model-value="handleSearch"
                :items="statusOptions"
                variant="outlined"
                density="compact"
              />
            </v-col>
            <v-col cols="12" md="2">
              <v-select
                v-model="filters.environment"
                label="Environment"
                clearable
                @update:model-value="handleSearch"
                :items="environmentOptions"
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

    <!-- CI Table -->
    <v-card class="table-card">
      <v-card-text>
        <v-data-table
          v-model:items-per-page="pagination.limit"
          :headers="tableHeaders"
          :items="cis"
          :loading="loading"
          :items-length="pagination.total"
          :page="pagination.page"
          @update:options="handleTableUpdate"
          class="elevation-1"
        >
          <template v-slot:item.name="{ item }">
            <v-btn
              variant="text"
              color="primary"
              @click="goToDetail(item.id)"
              class="text-left"
            >
              {{ item.name }}
            </v-btn>
          </template>
          
          <template v-slot:item.type="{ item }">
            <v-chip size="small" :color="getTypeColor(item.type)">
              {{ item.type }}
            </v-chip>
          </template>
          
          <template v-slot:item.status="{ item }">
            <v-chip size="small" :color="getStatusColor(item.status)">
              {{ item.status }}
            </v-chip>
          </template>
          
          <template v-slot:item.environment="{ item }">
            <v-chip size="small" :color="getEnvironmentColor(item.environment)">
              {{ item.environment }}
            </v-chip>
          </template>
          
          <template v-slot:item.schema_name="{ item }">
            <v-chip v-if="item.schema_name" size="small" color="secondary" variant="outlined">
              {{ item.schema_name }}
            </v-chip>
            <span v-else class="text-grey">N/A</span>
          </template>
          
          <template v-slot:item.created_at="{ item }">
            {{ formatDate(item.created_at) }}
          </template>
          
          <template v-slot:item.updated_at="{ item }">
            {{ formatDate(item.updated_at) }}
          </template>
          
          <template v-slot:item.actions="{ item }">
            <div class="action-buttons">
              <v-btn size="small" icon @click="goToEdit(item.id)" color="primary">
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
          <p>Are you sure you want to delete the configuration item "{{ deleteDialog.ci?.name }}"?</p>
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
import { ref, reactive, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useCIStore } from '@/stores/ci'
import { useSchemaStore } from '@/stores/schema'
import dayjs from 'dayjs'

const router = useRouter()
const ciStore = useCIStore()
const schemaStore = useSchemaStore()

// State
const loading = ref(false)
const cis = ref([])
const availableSchemas = ref([])
const filters = reactive({
  search: '',
  type: '',
  status: '',
  environment: '',
  schema_id: ''
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
  ci: null,
  loading: false
})

// Options for select fields
const typeOptions = [
  { title: 'Server', value: 'server' },
  { title: 'Database', value: 'database' },
  { title: 'Application', value: 'application' },
  { title: 'Network', value: 'network' },
  { title: 'Storage', value: 'storage' }
]

const statusOptions = [
  { title: 'Active', value: 'active' },
  { title: 'Inactive', value: 'inactive' },
  { title: 'Maintenance', value: 'maintenance' },
  { title: 'Decommissioned', value: 'decommissioned' }
]

const environmentOptions = [
  { title: 'Development', value: 'development' },
  { title: 'Testing', value: 'testing' },
  { title: 'Staging', value: 'staging' },
  { title: 'Production', value: 'production' }
]

const schemaOptions = computed(() => {
  return availableSchemas.value.map(schema => ({
    title: schema.name,
    value: schema.id
  }))
})

// Table headers
const tableHeaders = [
  { title: 'Name', key: 'name', sortable: true, width: '200px' },
  { title: 'Type', key: 'type', sortable: true, width: '120px' },
  { title: 'Status', key: 'status', sortable: true, width: '120px' },
  { title: 'Environment', key: 'environment', sortable: true, width: '120px' },
  { title: 'Schema', key: 'schema_name', sortable: true, width: '150px' },
  { title: 'Owner', key: 'owner', sortable: true, width: '150px' },
  { title: 'Created', key: 'created_at', sortable: true, width: '180px' },
  { title: 'Updated', key: 'updated_at', sortable: true, width: '180px' },
  { title: 'Actions', key: 'actions', sortable: false, width: '100px', align: 'end' }
]

// Methods
const fetchCIs = async () => {
  loading.value = true
  try {
    const params = {
      page: pagination.page,
      limit: pagination.limit,
      sort: pagination.sort,
      order: pagination.order,
      ...Object.fromEntries(Object.entries(filters).filter(([_, v]) => v !== ''))
    }
    
    const response = await ciStore.fetchCIs(params)
    cis.value = response.data || []
    pagination.total = response.total || 0
  } catch (error) {
    console.error('Failed to fetch CIs:', error)
  } finally {
    loading.value = false
  }
}

const fetchAvailableSchemas = async () => {
  try {
    const response = await schemaStore.fetchCiTypeSchemas({
      page: 1,
      page_size: 100,
      is_active: true
    })
    availableSchemas.value = response.schemas || []
  } catch (error) {
    console.error('Failed to fetch available schemas:', error)
  }
}

const handleSearch = () => {
  pagination.page = 1
  fetchCIs()
}

const resetFilters = () => {
  filters.search = ''
  filters.type = ''
  filters.status = ''
  filters.environment = ''
  filters.schema_id = ''
  pagination.page = 1
  fetchCIs()
}

const handleTableUpdate = (options) => {
  pagination.page = options.page
  pagination.limit = options.itemsPerPage
  pagination.sort = options.sortBy[0]?.key || 'created_at'
  pagination.order = options.sortBy[0]?.order === 'asc' ? 'asc' : 'desc'
  fetchCIs()
}

const goToCreate = () => {
  router.push('/cis/create')
}

const goToDetail = (id) => {
  router.push(`/cis/${id}`)
}

const goToEdit = (id) => {
  router.push(`/cis/${id}/edit`)
}

const confirmDelete = (ci) => {
  deleteDialog.ci = ci
  deleteDialog.visible = true
}

const deleteCI = async () => {
  if (!deleteDialog.ci) return
  
  deleteDialog.loading = true
  try {
    await ciStore.deleteCI(deleteDialog.ci.id)
    deleteDialog.visible = false
    fetchCIs()
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

const formatDate = (date) => {
  return dayjs(date).format('YYYY-MM-DD HH:mm:ss')
}

// Lifecycle
onMounted(async () => {
  await fetchAvailableSchemas()
  await fetchCIs()
})
</script>

<style scoped>
.cis-container {
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
