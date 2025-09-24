<template>
  <div class="graph-container">
    <div class="page-header">
      <h1>CI Relationship Graph</h1>
      <p>Visualize relationships between configuration items</p>
    </div>

    <!-- Graph Controls -->
    <div class="controls-section">
      <v-card>
        <v-card-text>
          <v-row>
            <v-col cols="12" md="3">
              <v-text-field
                v-model="searchQuery"
                label="Search CI..."
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
                v-model="selectedType"
                label="Filter by Type"
                clearable
                @update:model-value="handleTypeFilter"
                :items="typeOptions"
                variant="outlined"
                density="compact"
              />
            </v-col>
            <v-col cols="12" md="2">
              <v-select
                v-model="selectedEnvironment"
                label="Filter by Environment"
                clearable
                @update:model-value="handleEnvironmentFilter"
                :items="environmentOptions"
                variant="outlined"
                density="compact"
              />
            </v-col>
            <v-col cols="12" md="2">
              <v-select
                v-model="selectedRelationshipType"
                label="Relationship Type"
                clearable
                @update:model-value="handleRelationshipFilter"
                :items="relationshipTypeOptions"
                variant="outlined"
                density="compact"
              />
            </v-col>
            <v-col cols="12" md="2">
              <v-select
                v-model="selectedSchema"
                label="Schema"
                clearable
                @update:model-value="handleSchemaFilter"
                :items="schemaOptions"
                item-title="name"
                item-value="id"
                variant="outlined"
                density="compact"
              />
            </v-col>
            <v-col cols="12" md="1">
              <v-btn @click="resetFilters" variant="outlined" block :disabled="!hasActiveFilters">
                <v-icon>mdi-refresh</v-icon>
                Reset
              </v-btn>
            </v-col>
          </v-row>
        </v-card-text>
      </v-card>
    </div>

    <!-- Graph Display -->
    <v-card class="graph-card">
      <v-card-text>
        <div v-if="loading" class="loading-container">
          <v-skeleton-loader
            type="card"
            :loading="loading"
            class="mb-4"
          />
          <div class="loading-text">Loading graph data...</div>
        </div>
        
        <div v-else-if="graphError" class="error-container">
          <v-alert
            type="error"
            :title="graphError"
            prominent
          >
            <v-btn @click="loadGraphData" color="primary">Retry</v-btn>
          </v-alert>
        </div>
        
        <div v-else-if="graphData.nodes.length === 0" class="empty-container">
          <v-empty-state
            icon="mdi-graph"
            text="No configuration items found"
            action-text="Create CI"
            @click:action="router.push('/cis/create')"
          />
        </div>
        
        <div v-else class="graph-display">
          <div ref="graphElement" class="graph-element"></div>
          
          <!-- Graph Legend -->
          <div class="graph-legend">
            <h4>Legend</h4>
            <div class="legend-items">
              <div
                v-for="type in ciTypes"
                :key="type"
                class="legend-item"
                @click="highlightByType(type)"
              >
                <div
                  class="legend-color"
                  :style="{ backgroundColor: getNodeColor(type) }"
                ></div>
                <span class="legend-label">{{ formatType(type) }}</span>
              </div>
            </div>
            
            <div class="legend-divider"></div>
            
            <h4>Relationship Types</h4>
            <div class="legend-items">
              <div
                v-for="relType in relationshipTypes"
                :key="relType"
                class="legend-item"
                @click="highlightByRelationshipType(relType)"
              >
                <div
                  class="legend-color"
                  :style="{ backgroundColor: getEdgeColor(relType) }"
                ></div>
                <span class="legend-label">{{ formatRelationshipType(relType) }}</span>
              </div>
            </div>
          </div>
          
          <!-- Graph Info Panel -->
          <div class="graph-info">
            <h4>Graph Information</h4>
            <div class="info-stats">
              <div class="info-stat">
                <span class="stat-label">Total Nodes:</span>
                <span class="stat-value">{{ graphData.nodes.length }}</span>
              </div>
              <div class="info-stat">
                <span class="stat-label">Total Edges:</span>
                <span class="stat-value">{{ graphData.edges.length }}</span>
              </div>
              <div class="info-stat">
                <span class="stat-label">Selected Node:</span>
                <span class="stat-value">{{ selectedNode ? selectedNode.name : 'None' }}</span>
              </div>
            </div>
            
            <div v-if="selectedNode" class="node-details">
              <h5>Node Details</h5>
              <v-list density="compact">
                <v-list-item>
                  <v-list-item-title>Name</v-list-item-title>
                  <v-list-item-subtitle>{{ selectedNode.name }}</v-list-item-subtitle>
                </v-list-item>
                <v-list-item>
                  <v-list-item-title>Type</v-list-item-title>
                  <v-list-item-subtitle>
                    <v-chip size="small" :color="getNodeColor(selectedNode.type)">
                      {{ formatType(selectedNode.type) }}
                    </v-chip>
                  </v-list-item-subtitle>
                </v-list-item>
                <v-list-item>
                  <v-list-item-title>Environment</v-list-item-title>
                  <v-list-item-subtitle>
                    <v-chip size="small" :color="getEnvironmentColor(selectedNode.environment)">
                      {{ selectedNode.environment }}
                    </v-chip>
                  </v-list-item-subtitle>
                </v-list-item>
                <v-list-item>
                  <v-list-item-title>Status</v-list-item-title>
                  <v-list-item-subtitle>
                    <v-chip size="small" :color="getStatusColor(selectedNode.status)">
                      {{ selectedNode.status }}
                    </v-chip>
                  </v-list-item-subtitle>
                </v-list-item>
                <v-list-item>
                  <v-list-item-title>Owner</v-list-item-title>
                  <v-list-item-subtitle>{{ selectedNode.owner || 'N/A' }}</v-list-item-subtitle>
                </v-list-item>
              </v-list>
              <div class="node-actions">
                <v-btn size="small" @click="viewNodeDetails" color="primary">
                  <v-icon small>mdi-eye</v-icon>
                  View Details
                </v-btn>
                <v-btn size="small" @click="editNode" color="warning">
                  <v-icon small>mdi-pencil</v-icon>
                  Edit
                </v-btn>
              </div>
            </div>
          </div>
        </div>
      </v-card-text>
    </v-card>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, onUnmounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useCIStore } from '@/stores/ci'
import { useRelationshipStore } from '@/stores/relationship'
import { useSchemaStore } from '@/stores/schema'
import * as echarts from 'echarts'

const router = useRouter()
const ciStore = useCIStore()
const relationshipStore = useRelationshipStore()
const schemaStore = useSchemaStore()

// State
const loading = ref(false)
const graphError = ref('')
const graphElement = ref(null)
let graphInstance = null

// Filters
const searchQuery = ref('')
const selectedType = ref('')
const selectedEnvironment = ref('')
const selectedRelationshipType = ref('')
const selectedSchema = ref('')

// Graph data
const graphData = reactive({
  nodes: [],
  edges: []
})

const originalGraphData = reactive({
  nodes: [],
  edges: []
})

// Selected node
const selectedNode = ref(null)

// Available options
const availableSchemas = ref([])
const availableRelationshipTypes = ref([])

// Options
const ciTypes = ['server', 'database', 'application', 'network', 'storage']
const environments = ['development', 'testing', 'staging', 'production']

// Computed
const hasActiveFilters = computed(() => {
  return searchQuery.value || selectedType.value || selectedEnvironment.value || selectedRelationshipType.value || selectedSchema.value
})

const typeOptions = computed(() => {
  return ciTypes.map(type => ({
    title: formatType(type),
    value: type
  }))
})

const environmentOptions = computed(() => {
  return environments.map(env => ({
    title: env,
    value: env
  }))
})

const relationshipTypeOptions = computed(() => {
  return availableRelationshipTypes.value.map(type => ({
    title: formatRelationshipType(type),
    value: type
  }))
})

const schemaOptions = computed(() => {
  return availableSchemas.value.map(schema => ({
    title: schema.name,
    value: schema.id
  }))
})

const relationshipTypes = computed(() => {
  // Extract unique relationship types from graph data
  const types = new Set()
  graphData.edges.forEach(edge => {
    if (edge.relationship_type) {
      types.add(edge.relationship_type)
    }
  })
  return Array.from(types)
})

// Methods
const loadGraphData = async () => {
  loading.value = true
  graphError.value = ''
  
  try {
    const response = await relationshipStore.fetchCIRelationships({
      search: searchQuery.value,
      type: selectedType.value,
      environment: selectedEnvironment.value,
      relationship_type: selectedRelationshipType.value,
      schema_id: selectedSchema.value
    })
    
    const data = response.data || { nodes: [], edges: [] }
    
    // Transform data for ECharts
    graphData.nodes = data.nodes.map(node => ({
      id: node.id,
      name: node.name,
      type: node.type,
      environment: node.environment,
      status: node.status,
      owner: node.owner,
      symbolSize: getNodeSize(node.type),
      itemStyle: {
        color: getNodeColor(node.type)
      }
    }))
    
    graphData.edges = data.edges.map(edge => ({
      source: edge.source_id,
      target: edge.target_id,
      relationship_type: edge.relationship_type,
      schema_name: edge.schema_name,
      lineStyle: {
        color: getEdgeColor(edge.relationship_type),
        width: 2
      }
    }))
    
    // Store original data
    originalGraphData.nodes = [...graphData.nodes]
    originalGraphData.edges = [...graphData.edges]
    
    renderGraph()
  } catch (error) {
    console.error('Failed to load graph data:', error)
    graphError.value = error.message || 'Failed to load graph data'
  } finally {
    loading.value = false
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
    
    // Extract relationship types from schemas
    const types = new Set()
    response.schemas.forEach(schema => {
      if (schema.name) {
        types.add(schema.name)
      }
    })
    availableRelationshipTypes.value = Array.from(types)
  } catch (error) {
    console.error('Failed to load available schemas:', error)
  }
}

const renderGraph = () => {
  if (!graphElement.value) return
  
  if (graphInstance) {
    graphInstance.dispose()
  }
  
  graphInstance = echarts.init(graphElement.value)
  
  const option = {
    tooltip: {
      trigger: 'item',
      formatter: function(params) {
        if (params.dataType === 'node') {
          return `
            <div style="font-weight: bold; margin-bottom: 5px;">${params.data.name}</div>
            <div>Type: ${formatType(params.data.type)}</div>
            <div>Environment: ${params.data.environment}</div>
            <div>Status: ${params.data.status}</div>
            <div>Owner: ${params.data.owner || 'N/A'}</div>
          `
        } else if (params.dataType === 'edge') {
          return `
            <div>Relationship: ${params.data.relationship_type}</div>
            <div>Schema: ${params.data.schema_name || 'N/A'}</div>
            <div>${params.data.source} → ${params.data.target}</div>
          `
        }
      }
    },
    series: [{
      type: 'graph',
      layout: 'force',
      data: graphData.nodes,
      links: graphData.edges,
      roam: true,
      draggable: true,
      label: {
        show: true,
        position: 'right',
        formatter: '{b}'
      },
      lineStyle: {
        opacity: 0.9,
        width: 2,
        curveness: 0.1
      },
      emphasis: {
        focus: 'adjacency',
        lineStyle: {
          width: 4
        }
      },
      force: {
        repulsion: 1000,
        edgeLength: 200,
        gravity: 0.1,
        friction: 0.6
      }
    }]
  }
  
  graphInstance.setOption(option)
  
  // Add click event listener
  graphInstance.on('click', handleNodeClick)
}

const handleNodeClick = (params) => {
  if (params.dataType === 'node') {
    selectedNode.value = params.data
  }
}

const getNodeSize = (type) => {
  const sizes = {
    server: 60,
    database: 50,
    application: 55,
    network: 45,
    storage: 40
  }
  return sizes[type] || 50
}

const getNodeColor = (type) => {
  const colors = {
    server: '#1976d2',
    database: '#388e3c',
    application: '#f57c00',
    network: '#d32f2f',
    storage: '#757575'
  }
  return colors[type] || '#606266'
}

const getEdgeColor = (relationshipType) => {
  const colors = {
    'depends_on': '#1976d2',
    'connected_to': '#388e3c',
    'hosted_on': '#f57c00',
    'part_of': '#d32f2f',
    'related_to': '#757575'
  }
  return colors[relationshipType] || '#606266'
}

const getEnvironmentColor = (environment) => {
  const colors = {
    development: 'primary',
    testing: 'info',
    staging: 'warning',
    production: 'error'
  }
  return colors[environment] || 'secondary'
}

const getStatusColor = (status) => {
  const colors = {
    active: 'success',
    inactive: 'info',
    maintenance: 'warning',
    decommissioned: 'error'
  }
  return colors[status] || 'secondary'
}

const handleSearch = () => {
  if (!searchQuery.value) {
    resetFilters()
    return
  }
  
  const query = searchQuery.value.toLowerCase()
  graphData.nodes = originalGraphData.nodes.filter(node => 
    node.name.toLowerCase().includes(query) || 
    node.id.toString().includes(query)
  )
  
  updateGraphEdges()
  renderGraph()
}

const handleTypeFilter = () => {
  if (!selectedType.value) {
    resetFilters()
    return
  }
  
  graphData.nodes = originalGraphData.nodes.filter(node => 
    node.type === selectedType.value
  )
  
  updateGraphEdges()
  renderGraph()
}

const handleEnvironmentFilter = () => {
  if (!selectedEnvironment.value) {
    resetFilters()
    return
  }
  
  graphData.nodes = originalGraphData.nodes.filter(node => 
    node.environment === selectedEnvironment.value
  )
  
  updateGraphEdges()
  renderGraph()
}

const handleRelationshipFilter = () => {
  if (!selectedRelationshipType.value) {
    resetFilters()
    return
  }
  
  graphData.edges = originalGraphData.edges.filter(edge => 
    edge.relationship_type === selectedRelationshipType.value
  )
  
  updateGraphNodes()
  renderGraph()
}

const handleSchemaFilter = () => {
  if (!selectedSchema.value) {
    resetFilters()
    return
  }
  
  graphData.edges = originalGraphData.edges.filter(edge => 
    edge.schema_name === availableSchemas.value.find(s => s.id === selectedSchema.value)?.name
  )
  
  updateGraphNodes()
  renderGraph()
}

const updateGraphEdges = () => {
  const nodeIds = new Set(graphData.nodes.map(node => node.id))
  graphData.edges = originalGraphData.edges.filter(edge => 
    nodeIds.has(edge.source) && nodeIds.has(edge.target)
  )
}

const updateGraphNodes = () => {
  const nodeIds = new Set()
  graphData.edges.forEach(edge => {
    nodeIds.add(edge.source)
    nodeIds.add(edge.target)
  })
  
  graphData.nodes = originalGraphData.nodes.filter(node => 
    nodeIds.has(node.id)
  )
}

const resetFilters = () => {
  searchQuery.value = ''
  selectedType.value = ''
  selectedEnvironment.value = ''
  selectedRelationshipType.value = ''
  selectedSchema.value = ''
  
  graphData.nodes = [...originalGraphData.nodes]
  graphData.edges = [...originalGraphData.edges]
  
  renderGraph()
}

const highlightByType = (type) => {
  if (!graphInstance) return
  
  const nodes = graphData.nodes.map(node => ({
    ...node,
    itemStyle: {
      color: node.type === type ? getNodeColor(type) : '#ddd',
      opacity: node.type === type ? 1 : 0.3
    }
  }))
  
  graphInstance.setOption({
    series: [{
      data: nodes
    }]
  })
}

const highlightByRelationshipType = (relationshipType) => {
  if (!graphInstance) return
  
  const edges = graphData.edges.map(edge => ({
    ...edge,
    lineStyle: {
      ...edge.lineStyle,
      color: edge.relationship_type === relationshipType ? getEdgeColor(relationshipType) : '#ddd',
      opacity: edge.relationship_type === relationshipType ? 1 : 0.3
    }
  }))
  
  graphInstance.setOption({
    series: [{
      links: edges
    }]
  })
}

const viewNodeDetails = () => {
  if (selectedNode.value) {
    router.push(`/cis/${selectedNode.value.id}`)
  }
}

const editNode = () => {
  if (selectedNode.value) {
    router.push(`/cis/${selectedNode.value.id}/edit`)
  }
}

const formatType = (type) => {
  return type.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase())
}

const formatRelationshipType = (type) => {
  return type.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase())
}

// Handle window resize
const handleResize = () => {
  if (graphInstance) {
    graphInstance.resize()
  }
}

// Lifecycle
onMounted(async () => {
  await loadAvailableSchemas()
  await loadGraphData()
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
  if (graphInstance) {
    graphInstance.dispose()
  }
})
</script>

<style scoped>
.graph-container {
  padding: 20px;
}

.graph-header {
  margin-bottom: 20px;
}

.graph-header h1 {
  margin: 0 0 5px 0;
  font-size: 28px;
  color: #303133;
}

.graph-header p {
  margin: 0;
  color: #909399;
  font-size: 16px;
}

.controls-section {
  margin-bottom: 20px;
}

.graph-card {
  height: calc(100vh - 280px);
  min-height: 600px;
}

.loading-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
}

.loading-text {
  margin-top: 20px;
  color: #909399;
}

.error-container,
.empty-container {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
}

.graph-display {
  display: flex;
  height: 100%;
  position: relative;
}

.graph-element {
  flex: 1;
  height: 100%;
}

.graph-legend {
  position: absolute;
  top: 20px;
  right: 20px;
  background: white;
  border: 1px solid #ebeef5;
  border-radius: 4px;
  padding: 15px;
  box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.1);
  z-index: 1000;
  max-width: 200px;
  max-height: 400px;
  overflow-y: auto;
}

.graph-legend h4 {
  margin: 0 0 10px 0;
  font-size: 14px;
  color: #303133;
}

.legend-items {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.legend-item {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  padding: 4px;
  border-radius: 4px;
  transition: background-color 0.2s;
}

.legend-item:hover {
  background-color: #f5f7fa;
}

.legend-color {
  width: 16px;
  height: 16px;
  border-radius: 2px;
}

.legend-label {
  font-size: 12px;
  color: #606266;
  text-transform: capitalize;
}

.legend-divider {
  height: 1px;
  background: #e0e0e0;
  margin: 15px 0;
}

.graph-info {
  position: absolute;
  bottom: 20px;
  left: 20px;
  background: white;
  border: 1px solid #ebeef5;
  border-radius: 4px;
  padding: 15px;
  box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.1);
  z-index: 1000;
  max-width: 300px;
  max-height: 400px;
  overflow-y: auto;
}

.graph-info h4 {
  margin: 0 0 10px 0;
  font-size: 14px;
  color: #303133;
}

.info-stats {
  margin-bottom: 15px;
}

.info-stat {
  display: flex;
  justify-content: space-between;
  margin-bottom: 5px;
}

.stat-label {
  font-size: 12px;
  color: #909399;
}

.stat-value {
  font-size: 12px;
  color: #303133;
  font-weight: 500;
}

.node-details h5 {
  margin: 0 0 10px 0;
  font-size: 13px;
  color: #303133;
  border-bottom: 1px solid #ebeef5;
  padding-bottom: 5px;
}

.node-actions {
  margin-top: 10px;
  display: flex;
  gap: 8px;
}

/* Custom scrollbar for graph info */
.graph-legend::-webkit-scrollbar,
.graph-info::-webkit-scrollbar {
  width: 6px;
}

.graph-legend::-webkit-scrollbar-track,
.graph-info::-webkit-scrollbar-track {
  background: #f1f1f1;
  border-radius: 3px;
}

.graph-legend::-webkit-scrollbar-thumb,
.graph-info::-webkit-scrollbar-thumb {
  background: #c1c1c1;
  border-radius: 3px;
}

.graph-legend::-webkit-scrollbar-thumb:hover,
.graph-info::-webkit-scrollbar-thumb:hover {
  background: #a8a8a8;
}

.v-chip {
  text-transform: capitalize;
}
</style>
