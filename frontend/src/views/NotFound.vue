<template>
  <div class="not-found-container">
    <el-result
      icon="error"
      title="404 - Page Not Found"
      sub-title="The page you're looking for doesn't exist or has been moved."
    >
      <template #extra>
        <el-button type="primary" @click="goToDashboard">
          Go to Dashboard
        </el-button>
        <el-button @click="goBack">
          Go Back
        </el-button>
      </template>
    </el-result>
  </div>
</template>

<script setup>
import { useRouter } from 'vue-router'

const router = useRouter()

const goToDashboard = () => {
  router.push('/')
}

const goBack = () => {
  router.go(-1)
}
</script>

<style scoped>
.not-found-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f5f7fa;
}
</style>
