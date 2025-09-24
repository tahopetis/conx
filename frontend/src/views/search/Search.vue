<template>
  <div class="search-container">
    <div class="search-header">
      <h1>Advanced Search</h1>
      <p>Search and filter configuration items</p>
    </div>

    <!-- Search Form -->
    <el-card class="search-form-card">
      <el-form
        ref="searchFormRef"
        :model="searchForm"
        label-width="120px"
        label-position="left"
      >
        <el-row :gutter="20">
          <el-col :span="8">
            <el-form-item label="Keyword Search">
              <el-input
                v-model="searchForm.keyword"
                placeholder="Search by name, description, etc."
                clearable
                @keyup.enter="handleSearch"
              >
                <template #prefix>
                  <el-icon><Search /></el-icon>
                </template>
              </el-input>
            </el-form-item>
          </el-col>
          <el-col :span="6">
            <el-form-item label="Type">
              <el-select
                v-model="searchForm.type"
                placeholder="All Types"
                clearable
                @change="handleSearch"
              >
                <el-option
                  v-for="type in ciTypes"
                  :key="type"
                  :label="type"
                  :value="type"
                />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="6">
            <el-form-item label="Environment">
              <el-select
                v-model="searchForm.environment"
                placeholder="All Environments"
                clearable
                @change="handleSearch"
              >
                <el-option
                  v-for="env in environments"
                  :key="env"
                  :label="env"
                  :value="env"
                />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="4">
            <el-form-item label="Status">
              <el-select
                v-model="searchForm.status"
                placeholder="All Status"
                clearable
                @change="handleSearch"
              >
                <el-option
                  v-for="status in statuses"
                  :key="status"
                  :label="status"
                  :value="status"
                />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="20">
          <el-col :span="8">
            <el-form-item label="Owner">
              <el-input
                v-model="searchForm.owner"
                placeholder="Filter by owner"
                clearable
                @keyup.enter="handleSearch"
              />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="Team">
              <el-input
                v-model="searchForm.team"
                placeholder="Filter by team"
                clearable
                @keyup.enter="handleSearch"
              />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="Business Unit">
              <el-input
                v-model="searchForm.business_unit"
                placeholder="Filter by business unit"
                clearable
                @keyup.enter="handleSearch"
              />
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="20">
          <el-col :span="8">
            <el-form-item label="IP Address">
              <el-input
                v-model="searchForm.ip_address"
                placeholder="Filter by IP address"
                clearable
                @keyup.enter="handleSearch"
              />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="Hostname">
              <el-input
                v-model="searchForm.hostname"
                placeholder="Filter by hostname"
                clearable
                @keyup.enter="handleSearch"
              />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="Location">
              <el-input
                v-model="searchForm.location"
                placeholder="Filter by location"
                clearable
                @keyup.enter="handleSearch"
              />
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item label="Date Range">
              <el-date-picker
                v-model="searchForm.date_range"
                type="daterange"
                range-separator="to"
                start-placeholder="Start date"
                end-placeholder="End date"
                format="YYYY-MM-DD"
                value-format="YYYY-MM-DD"
                @change="handleSearch"
              />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="Actions">
              <el-button-group>
                <el-button type="primary" @click="handleSearch" :loading="loading">
                  <el-icon><Search /></el-icon>
                  Search
                </el-button>
                <el-button @click="resetSearch">
                  <el-icon><RefreshRight /></el-icon>
                  Reset
                </el-button>
                <el-button @click="saveSearch" :disabled="!hasSearchCriteria">
                  <el-icon><Star /></el-icon>
                  Save Search
                </el-button>
              </el-button-group>
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
    </el-card>

    <!-- Saved Searches -->
    <el-card v-if="savedSearches.length > 0" class="saved-searches-card">
      <template #header>
        <div class="card-header">
          <h3>Saved Searches</h3>
          <el-button size="small" @click="showSavedSearches = !showSavedSearches">
            {{ showSavedSearches ? 'Hide' : 'Show' }}
          </el-button>
        </div>
      </template>
      
      <div v-show="showSavedSearches" class="saved-searches-list">
        <el-tag
          v-for="search in savedSearches"
          :key="search.id"
          closable
          @click="loadSavedSearch(search)"
          @close="deleteSavedSearch(search.id)"
          class="saved-search-tag"
        >
          {{ search.name }}
        </el-tag>
      </div>
    </el-card>

    <!-- Search Results -->
    <el-card class="results-card">
      <template #header>
        <div class="card-header">
          <h3>Search Results</h3>
          <div class="results-info">
            <span v-if="!loading">Found {{ pagination.total }} results</span>
            <el-button-group size="small">
              <el-button @click="exportResults" :loading="exporting" :disabled="results.length === 0">
                <el-icon><Download /></el-icon>
                Export
              </el-button>
              <el-button @click="toggleViewMode">
                <el-icon><Grid v-if="viewMode === 'table'" /><List v-else /></el-icon>
                {{ viewMode === 'table' ? 'Cards' : 'Table' }}
              </el-button>
            </el-button-group>
          </div>
        </div>
      </template>

      <!-- Loading State -->
      <div v-if="loading" class="loading-container">
        <el-skeleton :rows="5" animated />
      </div>

      <!-- Error State -->
      <div v-else-if="searchError" class="error-container">
        <el-result
          icon="error"
          title="Search Error"
          :sub-title="searchError"
        >
          <template #extra>
            <el-button type="primary" @click="handleSearch">
              Retry
            </el-button>
          </template>
        </el-result>
      </div>

      <!-- No Results State -->
      <div v-else-if="results.length === 0 && hasSearched" class="empty-container">
        <el-empty description="No results found">
          <el-button type="primary" @click="resetSearch">
            Clear Search
          </el-button>
        </el-empty>
      </div>

      <!-- Table View -->
      <div v-else-if="viewMode === 'table' && results.length > 0">
        <el-table
          :data="results"
          style="width: 100%"
          @row-click="handleRowClick"
        >
          <el-table-column prop="name" label="Name" min-width="200">
            <template #default="{ row }">
              <div class="table-cell-name">
                <el-icon :class="getIconClass(row.type)" />
                <span>{{ row.name }}</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column prop="type" label="Type" width="120">
            <template #default="{ row }">
              <el-tag :type="getTypeTagType(row.type)" size="small">
                {{ row.type }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="environment" label="Environment" width="120">
            <template #default="{ row }">
              <el-tag :type="getEnvironmentTagType(row.environment)" size="small">
                {{ row.environment }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="status" label="Status" width="120">
            <template #default="{ row }">
              <el-tag :type="getStatusTagType(row.status)" size="small">
                {{ row.status }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="owner" label="Owner" width="150" />
          <el-table-column prop="team" label="Team" width="150" />
          <el-table-column prop="updated_at" label="Updated" width="180">
            <template #default="{ row }">
              {{ formatDate(row.updated_at) }}
            </template>
          </el-table-column>
          <el-table-column label="Actions" width="150" fixed="right">
            <template #default="{ row }">
              <el-button-group size="small">
                <el-button @click.stop="viewDetails(row)">
                  <el-icon><View /></el-icon>
                </el-button>
                <el-button @click.stop="editCI(row)">
                  <el-icon><Edit /></el-icon>
                </el-button>
                <el-button @click.stop="viewInGraph(row)">
                  <el-icon><Share /></el-icon>
                </el-button>
              </el-button-group>
            </template>
          </el-table-column>
        </el-table>

        <!-- Pagination -->
        <div class="pagination-container">
          <el-pagination
            v-model:current-page="pagination.page"
            v-model:page-size="pagination.limit"
            :page-sizes="[10, 20, 50, 100]"
            :total="pagination.total"
            layout="total, sizes, prev, pager, next, jumper"
            @size-change="handleSizeChange"
            @current-change="handlePageChange"
          />
        </div>
      </div>

      <!-- Card View -->
      <div v-else-if="viewMode === 'cards' && results.length > 0" class="cards-view">
        <el-row :gutter="20">
          <el-col
            v-for="item in results"
            :key="item.id"
            :span="8"
            class="card-col"
          >
            <el-card class="result-card" shadow="hover" @click="viewDetails(item)">
              <template #header>
                <div class="card-header">
                  <div class="card-title">
                    <el-icon :class="getIconClass(item.type)" />
                    <span>{{ item.name }}</span>
                  </div>
                  <div class="card-actions">
                    <el-button size="small" text @click.stop="editCI(item)">
                      <el-icon><Edit /></el-icon>
                    </el-button>
                    <el-button size="small" text @click.stop="viewInGraph(item)">
                      <el-icon><Share /></el-icon>
                    </el-button>
                  </div>
                </div>
              </template>
              
              <div class="card-content">
                <div class="card-info">
                  <div class="info-item">
                    <span class="info-label">Type:</span>
                    <el-tag :type="getTypeTagType(item.type)" size="small">
                      {{ item.type }}
                    </el-tag>
                  </div>
                  <div class="info-item">
                    <span class="info-label">Environment:</span>
                    <el-tag :type="getEnvironmentTagType(item.environment)" size="small">
                      {{ item.environment }}
                    </el-tag>
                  </div>
                  <div class="info-item">
                    <span class="info-label">Status:</span>
                    <el-tag :type="getStatusTagType(item.status)" size="small">
                      {{ item.status }}
                    </el-tag>
                  </div>
                  <div class="info-item">
                    <span class="info-label">Owner:</span>
                    <span class="info-value">{{ item.owner }}</span>
                  </div>
                  <div class="info-item">
                    <span class="info-label">Team:</span>
                    <span class="info-value">{{ item.team }}</span>
                  </div>
                  <div v-if="item.description" class="info-item">
                    <span class="info-label">Description:</span>
                    <span class="info-value description">{{ item.description }}</span>
                  </div>
                </div>
              </div>
            </el-card>
          </el-col>
        </el-row>

        <!-- Pagination for Cards -->
        <div class="pagination-container">
          <el-pagination
            v-model:current-page="pagination.page"
            v-model:page-size="pagination.limit"
            :page-sizes="[9, 18, 27, 36]"
            :total="pagination.total"
            layout="total, sizes, prev, pager, next"
            @size-change="handleSizeChange"
            @current-change="handlePageChange"
          />
        </div>
      </div>
    </el-card>

    <!-- Save Search Dialog -->
    <el-dialog
      v-model="showSaveDialog"
      title="Save Search"
      width="400px"
    >
      <el-form :model="saveSearchForm" label-width="80px">
        <el-form-item label="Name" required>
          <el-input
            v-model="saveSearchForm.name"
            placeholder="Enter search name"
          />
        </el-form-item>
        <el-form-item label="Description">
          <el-input
            v-model="saveSearchForm.description"
            type="textarea"
            :rows="2"
            placeholder="Enter description (optional)"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showSaveDialog = false">Cancel</el-button>
        <el-button type="primary" @click="confirmSaveSearch" :loading="saving">
          Save
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { 
  Search, 
  RefreshRight, 
  Star, 
  Download, 
  Grid, 
  List,
  View,
  Edit,
  Share,
  Monitor,
  DataBase,
  Coin,
  Connection,
  Box
} from '@element-plus/icons-vue'
import api from '@/services/api'
import dayjs from 'dayjs'

const router = useRouter()

// State
const loading = ref(false)
const exporting = ref(false)
const saving = ref(false)
const searchError = ref('')
const hasSearched = ref(false)
const viewMode = ref('table')
const showSavedSearches = ref(true)
const showSaveDialog = ref(false)

// Search form
const searchForm = reactive({
  keyword: '',
  type: '',
  environment: '',
  status: '',
  owner: '',
  team: '',
  business_unit: '',
  ip_address: '',
  hostname: '',
  location: '',
  date_range: []
})

// Pagination
const pagination = reactive({
  page: 1,
  limit: 20,
  total: 0
})

// Results
const results = ref([])

// Saved searches
const savedSearches = ref([])

// Save search form
const saveSearchForm = reactive({
  name: '',
  description: ''
})

// Options
const ciTypes = ['server', 'database', 'application', 'network', 'storage']
const environments = ['development', 'testing', 'staging', 'production']
const statuses = ['active', 'inactive', 'maintenance', 'decommissioned']

// Computed
const hasSearchCriteria = computed(() => {
  return Object.values(searchForm).some(value => {
    if (Array.isArray(value)) {
      return value.length > 0
    }
    return value !== '' && value !== null
  })
})

// Methods
const handleSearch = async () => {
  loading.value = true
  searchError.value = ''
  hasSearched.value = true
  
  try {
    const params = {
      ...searchForm,
      page: pagination.page,
      limit: pagination.limit
    }
    
    // Remove empty values
    Object.keys(params).forEach(key => {
      if (params[key] === '' || params[key] === null || (Array.isArray(params[key]) && params[key].length === 0)) {
        delete params[key]
      }
    })
    
    const response = await api.get('/cis/search', { params })
    results.value = response.data.data || []
    pagination.total = response.data.total || 0
  } catch (error) {
    console.error('Search failed:', error)
    searchError.value = error.response?.data?.message || 'Search failed'
  } finally {
    loading.value = false
  }
}

const resetSearch = () => {
  Object.keys(searchForm).forEach(key => {
    if (Array.isArray(searchForm[key])) {
      searchForm[key] = []
    } else {
      searchForm[key] = ''
    }
  })
  
  pagination.page = 1
  results.value = []
  searchError.value = ''
  hasSearched.value = false
}

const handleSizeChange = (size) => {
  pagination.limit = size
  pagination.page = 1
  handleSearch()
}

const handlePageChange = (page) => {
  pagination.page = page
  handleSearch()
}

const toggleViewMode = () => {
  viewMode.value = viewMode.value === 'table' ? 'cards' : 'table'
}

const exportResults = async () => {
  if (results.value.length === 0) return
  
  exporting.value = true
  
  try {
    const params = {
      ...searchForm,
      format: 'csv'
    }
    
    // Remove empty values
    Object.keys(params).forEach(key => {
      if (params[key] === '' || params[key] === null || (Array.isArray(params[key]) && params[key].length === 0)) {
        delete params[key]
      }
    })
    
    const response = await api.get('/cis/search/export', { 
      params,
      responseType: 'blob'
    })
    
    const url = window.URL.createObjectURL(new Blob([response.data]))
    const link = document.createElement('a')
    link.href = url
    link.download = `ci-search-results-${new Date().toISOString().split('T')[0]}.csv`
    link.click()
    window.URL.revokeObjectURL(url)
    
    ElMessage.success('Results exported successfully')
  } catch (error) {
    console.error('Export failed:', error)
    ElMessage.error('Failed to export results')
  } finally {
    exporting.value = false
  }
}

const saveSearch = () => {
  saveSearchForm.name = ''
  saveSearchForm.description = ''
  showSaveDialog.value = true
}

const confirmSaveSearch = async () => {
  if (!saveSearchForm.name.trim()) {
    ElMessage.error('Please enter a search name')
    return
  }
  
  saving.value = true
  
  try {
    const searchData = {
      name: saveSearchForm.name,
      description: saveSearchForm.description,
      criteria: searchForm
    }
    
    await api.post('/user/saved-searches', searchData)
    
    ElMessage.success('Search saved successfully')
    showSaveDialog.value = false
    loadSavedSearches()
  } catch (error) {
    console.error('Failed to save search:', error)
    ElMessage.error('Failed to save search')
  } finally {
    saving.value = false
  }
}

const loadSavedSearches = async () => {
  try {
    const response = await api.get('/user/saved-searches')
    savedSearches.value = response.data.data || []
  } catch (error) {
    console.error('Failed to load saved searches:', error)
  }
}

const loadSavedSearch = (search) => {
  Object.assign(searchForm, search.criteria)
  handleSearch()
}

const deleteSavedSearch = async (id) => {
  try {
    await ElMessageBox.confirm(
      'Are you sure you want to delete this saved search?',
      'Delete Saved Search',
      {
        confirmButtonText: 'Delete',
        cancelButtonText: 'Cancel',
        type: 'warning'
      }
    )
    
    await api.delete(`/user/saved-searches/${id}`)
    
    ElMessage.success('Saved search deleted successfully')
    loadSavedSearches()
  } catch (error) {
    if (error !== 'cancel') {
      console.error('Failed to delete saved search:', error)
      ElMessage.error('Failed to delete saved search')
    }
  }
}

const viewDetails = (ci) => {
  router.push(`/cis/${ci.id}`)
}

const editCI = (ci) => {
  router.push(`/cis/${ci.id}/edit`)
}

const viewInGraph = (ci) => {
  router.push({
    path: '/graph',
    query: { highlight: ci.id }
  })
}

const handleRowClick = (row) => {
  viewDetails(row)
}

const getIconClass = (type) => {
  const icons = {
    server: 'server-icon',
    database: 'database-icon',
    application: 'application-icon',
    network: 'network-icon',
    storage: 'storage-icon'
  }
  return icons[type] || 'default-icon'
}

const getTypeTagType = (type) => {
  const types = {
    server: '',
    database: 'success',
    application: 'warning',
    network: 'info',
    storage: 'danger'
  }
  return types[type] || ''
}

const getEnvironmentTagType = (environment) => {
  const environments = {
    development: '',
    staging: 'warning',
    production: 'danger',
    testing: 'info'
  }
  return environments[environment] || ''
}

const getStatusTagType = (status) => {
  const statuses = {
    active: 'success',
    inactive: 'info',
    maintenance: 'warning',
    decommissioned: 'danger'
  }
  return statuses[status] || ''
}

const formatDate = (date) => {
  return dayjs(date).format('YYYY-MM-DD HH:mm:ss')
}

// Lifecycle
onMounted(async () => {
  await loadSavedSearches()
  
  // Check for search parameters in URL
  const query = router.currentRoute.value.query
  if (query.search) {
    try {
      const searchParams = JSON.parse(query.search)
      Object.assign(searchForm, searchParams)
      handleSearch()
    } catch (error) {
      console.error('Failed to parse search parameters:', error)
    }
  }
})
</script>

<style scoped>
.search-container {
  padding: 20px;
}

.search-header {
  margin-bottom: 20px;
}

.search-header h1 {
  margin: 0 0 5px 0;
  font-size: 28px;
  color: #303133;
}

.search-header p {
  margin: 0;
  color: #909399;
  font-size: 16px;
}

.search-form-card {
  margin-bottom: 20px;
}

.saved-searches-card {
  margin-bottom: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.card-header h3 {
  margin: 0;
  font-size: 16px;
  color: #303133;
}

.saved-searches-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.saved-search-tag {
  cursor: pointer;
}

.results-card {
  margin-bottom: 20px;
}

.results-info {
  display: flex;
  align-items: center;
  gap: 15px;
}

.loading-container,
.error-container,
.empty-container {
  padding: 40px 0;
  text-align: center;
}

.table-cell-name {
  display: flex;
  align-items: center;
  gap: 8px;
}

.pagination-container {
  display: flex;
  justify-content: center;
  margin-top: 20px;
}

.cards-view {
  margin-top: 20px;
}

.card-col {
  margin-bottom: 20px;
}

.result-card {
  cursor: pointer;
  transition: transform 0.2s, box-shadow 0.2s;
}

.result-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

.card-title {
  display: flex;
  align-items: center;
  gap: 8px;
}

.card-actions {
  display: flex;
  gap: 4px;
}

.card-content {
  padding: 0;
}

.card-info {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.info-item {
  display: flex;
  align-items: flex-start;
  gap: 8px;
}

.info-label {
  min-width: 80px;
  font-size: 12px;
  color: #909399;
  font-weight: 500;
}

.info-value {
  font-size: 12px;
  color: #606266;
  flex: 1;
}

.description {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* Icon styles */
.server-icon,
.database-icon,
.application-icon,
.network-icon,
.storage-icon {
  font-size: 16px;
}

.server-icon { color: #409EFF; }
.database-icon { color: #67C23A; }
.application-icon { color: #E6A23C; }
.network-icon { color: #F56C6C; }
.storage-icon { color: #909399; }
</style>
