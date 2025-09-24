<template>
  <div class="dashboard-container">
    <div class="dashboard-header">
      <h1>Dashboard</h1>
      <p>Welcome back, {{ authStore.user?.full_name || 'User' }}!</p>
    </div>

    <!-- Statistics Cards -->
    <el-row :gutter="20" class="stats-row">
      <el-col :span="6">
        <el-card class="stat-card" shadow="hover">
          <div class="stat-content">
            <div class="stat-icon total-cis">
              <el-icon><Monitor /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-number">{{ stats.totalCIs }}</div>
              <div class="stat-label">Total CIs</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card class="stat-card" shadow="hover">
          <div class="stat-content">
            <div class="stat-icon active-cis">
              <el-icon><Check /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-number">{{ stats.activeCIs }}</div>
              <div class="stat-label">Active CIs</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card class="stat-card" shadow="hover">
          <div class="stat-content">
            <div class="stat-icon relationships">
              <el-icon><Connection /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-number">{{ stats.relationships }}</div>
              <div class="stat-label">Relationships</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card class="stat-card" shadow="hover">
          <div class="stat-content">
            <div class="stat-icon recent-activity">
              <el-icon><Clock /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-number">{{ stats.recentActivity }}</div>
              <div class="stat-label">Recent Activity</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- Charts Row -->
    <el-row :gutter="20" class="charts-row">
      <el-col :span="12">
        <el-card class="chart-card">
          <template #header>
            <div class="card-header">
              <h3>CI Types Distribution</h3>
              <el-button size="small" text @click="refreshCharts">
                <el-icon><RefreshRight /></el-icon>
              </el-button>
            </div>
          </template>
          <div ref="ciTypesChart" class="chart-container"></div>
        </el-card>
      </el-col>
      <el-col :span="12">
        <el-card class="chart-card">
          <template #header>
            <div class="card-header">
              <h3>Environment Distribution</h3>
              <el-button size="small" text @click="refreshCharts">
                <el-icon><RefreshRight /></el-icon>
              </el-button>
            </div>
          </template>
          <div ref="environmentChart" class="chart-container"></div>
        </el-card>
      </el-col>
    </el-row>

    <!-- Recent Activity and Quick Actions -->
    <el-row :gutter="20" class="bottom-row">
      <el-col :span="16">
        <el-card class="activity-card">
          <template #header>
            <div class="card-header">
              <h3>Recent Activity</h3>
              <el-button size="small" text @click="goToSearch">View All</el-button>
            </div>
          </template>
          <el-timeline>
            <el-timeline-item
              v-for="activity in recentActivities"
              :key="activity.id"
              :timestamp="formatDate(activity.created_at)"
              :type="getActivityType(activity.event_type)"
            >
              <h4>{{ activity.event_type }}</h4>
              <p>{{ activity.description }}</p>
              <p v-if="activity.user" class="activity-user">By: {{ activity.user }}</p>
            </el-timeline-item>
          </el-timeline>
          <div v-if="recentActivities.length === 0" class="empty-state">
            <el-empty description="No recent activity" />
          </div>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card class="actions-card">
          <template #header>
            <h3>Quick Actions</h3>
          </template>
          <div class="quick-actions">
            <el-button type="primary" size="large" @click="goToCreateCI" class="action-button">
              <el-icon><Plus /></el-icon>
              Create CI
            </el-button>
            <el-button size="large" @click="goToCIs" class="action-button">
              <el-icon><List /></el-icon>
              View All CIs
            </el-button>
            <el-button size="large" @click="goToGraph" class="action-button">
              <el-icon><Share /></el-icon>
              View Graph
            </el-button>
            <el-button size="large" @click="goToSearch" class="action-button">
              <el-icon><Search /></el-icon>
              Search CIs
            </el-button>
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { 
  Monitor, 
  Check, 
  Connection, 
  Clock, 
  RefreshRight,
  Plus,
  List,
  Share,
  Search
} from '@element-plus/icons-vue'
import { useAuthStore } from '@/stores/auth'
import api from '@/services/api'
import dayjs from 'dayjs'
import * as echarts from 'echarts'

const router = useRouter()
const authStore = useAuthStore()

// State
const stats = reactive({
  totalCIs: 0,
  activeCIs: 0,
  relationships: 0,
  recentActivity: 0
})

const recentActivities = ref([])
const ciTypesChart = ref(null)
const environmentChart = ref(null)
let ciTypesChartInstance = null
let environmentChartInstance = null

// Methods
const fetchDashboardStats = async () => {
  try {
    const response = await api.get('/dashboard/stats')
    Object.assign(stats, response.data)
  } catch (error) {
    console.error('Failed to fetch dashboard stats:', error)
    ElMessage.error('Failed to load dashboard statistics')
  }
}

const fetchRecentActivities = async () => {
  try {
    const response = await api.get('/dashboard/recent-activity')
    recentActivities.value = response.data.data || []
  } catch (error) {
    console.error('Failed to fetch recent activities:', error)
  }
}

const fetchChartData = async () => {
  try {
    const [ciTypesResponse, environmentResponse] = await Promise.all([
      api.get('/dashboard/ci-types'),
      api.get('/dashboard/environments')
    ])

    renderCiTypesChart(ciTypesResponse.data)
    renderEnvironmentChart(environmentResponse.data)
  } catch (error) {
    console.error('Failed to fetch chart data:', error)
  }
}

const renderCiTypesChart = (data) => {
  if (!ciTypesChart.value) return

  if (ciTypesChartInstance) {
    ciTypesChartInstance.dispose()
  }

  ciTypesChartInstance = echarts.init(ciTypesChart.value)
  
  const option = {
    tooltip: {
      trigger: 'item',
      formatter: '{a} <br/>{b}: {c} ({d}%)'
    },
    legend: {
      orient: 'vertical',
      left: 'left'
    },
    series: [
      {
        name: 'CI Types',
        type: 'pie',
        radius: '50%',
        data: Object.entries(data).map(([name, value]) => ({ name, value })),
        emphasis: {
          itemStyle: {
            shadowBlur: 10,
            shadowOffsetX: 0,
            shadowColor: 'rgba(0, 0, 0, 0.5)'
          }
        }
      }
    ]
  }

  ciTypesChartInstance.setOption(option)
}

const renderEnvironmentChart = (data) => {
  if (!environmentChart.value) return

  if (environmentChartInstance) {
    environmentChartInstance.dispose()
  }

  environmentChartInstance = echarts.init(environmentChart.value)
  
  const option = {
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'shadow'
      }
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true
    },
    xAxis: {
      type: 'value'
    },
    yAxis: {
      type: 'category',
      data: Object.keys(data)
    },
    series: [
      {
        name: 'Environments',
        type: 'bar',
        data: Object.values(data),
        itemStyle: {
          color: function(params) {
            const colors = ['#409EFF', '#67C23A', '#E6A23C', '#F56C6C']
            return colors[params.dataIndex % colors.length]
          }
        }
      }
    ]
  }

  environmentChartInstance.setOption(option)
}

const refreshCharts = () => {
  fetchChartData()
}

const getActivityType = (eventType) => {
  const types = {
    created: 'success',
    updated: 'warning',
    deleted: 'danger',
    restored: 'info'
  }
  return types[eventType] || 'primary'
}

const formatDate = (date) => {
  return dayjs(date).format('YYYY-MM-DD HH:mm:ss')
}

const goToCreateCI = () => {
  router.push('/cis/create')
}

const goToCIs = () => {
  router.push('/cis')
}

const goToGraph = () => {
  router.push('/graph')
}

const goToSearch = () => {
  router.push('/search')
}

// Handle window resize
const handleResize = () => {
  if (ciTypesChartInstance) ciTypesChartInstance.resize()
  if (environmentChartInstance) environmentChartInstance.resize()
}

// Lifecycle
onMounted(async () => {
  await Promise.all([
    fetchDashboardStats(),
    fetchRecentActivities(),
    fetchChartData()
  ])

  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
  if (ciTypesChartInstance) ciTypesChartInstance.dispose()
  if (environmentChartInstance) environmentChartInstance.dispose()
})
</script>

<style scoped>
.dashboard-container {
  padding: 20px;
}

.dashboard-header {
  margin-bottom: 30px;
}

.dashboard-header h1 {
  margin: 0 0 5px 0;
  font-size: 28px;
  color: #303133;
}

.dashboard-header p {
  margin: 0;
  color: #909399;
  font-size: 16px;
}

.stats-row {
  margin-bottom: 20px;
}

.stat-card {
  cursor: pointer;
  transition: transform 0.2s;
}

.stat-card:hover {
  transform: translateY(-2px);
}

.stat-content {
  display: flex;
  align-items: center;
  gap: 15px;
}

.stat-icon {
  width: 50px;
  height: 50px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
  color: white;
}

.stat-icon.total-cis {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.stat-icon.active-cis {
  background: linear-gradient(135deg, #56ab2f 0%, #a8e6cf 100%);
}

.stat-icon.relationships {
  background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
}

.stat-icon.recent-activity {
  background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
}

.stat-info {
  flex: 1;
}

.stat-number {
  font-size: 24px;
  font-weight: bold;
  color: #303133;
  margin-bottom: 5px;
}

.stat-label {
  font-size: 14px;
  color: #909399;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.charts-row,
.bottom-row {
  margin-bottom: 20px;
}

.chart-card,
.activity-card,
.actions-card {
  height: 400px;
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

.chart-container {
  height: 320px;
  width: 100%;
}

.activity-card {
  overflow-y: auto;
}

.activity-user {
  margin: 5px 0 0 0;
  font-size: 12px;
  color: #909399;
  font-style: italic;
}

.empty-state {
  padding: 40px 0;
  text-align: center;
}

.quick-actions {
  display: flex;
  flex-direction: column;
  gap: 15px;
  height: calc(100% - 60px);
  justify-content: center;
}

.action-button {
  width: 100%;
  height: 50px;
  justify-content: flex-start;
  gap: 10px;
}

/* Custom scrollbar for activity card */
.activity-card::-webkit-scrollbar {
  width: 6px;
}

.activity-card::-webkit-scrollbar-track {
  background: #f1f1f1;
  border-radius: 3px;
}

.activity-card::-webkit-scrollbar-thumb {
  background: #c1c1c1;
  border-radius: 3px;
}

.activity-card::-webkit-scrollbar-thumb:hover {
  background: #a8a8a8;
}
</style>
