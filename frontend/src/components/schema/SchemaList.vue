<template>
  <div class="schema-list">
    <v-card>
      <v-card-title class="d-flex justify-space-between align-center">
        <span>Schema Management</span>
        <v-btn color="primary" @click="navigateToCreate">
          <v-icon left>mdi-plus</v-icon>
          Create Schema
        </v-btn>
      </v-card-title>

      <v-card-text>
        <!-- Schema Type Tabs -->
        <v-tabs v-model="activeTab" @change="handleTabChange">
          <v-tab>CI Type Schemas</v-tab>
          <v-tab>Relationship Type Schemas</v-tab>
        </v-tabs>

        <v-tabs-items v-model="activeTab">
          <v-tab-item>
            <ci-type-schemas ref="ciTypeSchemas" />
          </v-tab-item>
          <v-tab-item>
            <relationship-type-schemas ref="relationshipTypeSchemas" />
          </v-tab-item>
        </v-tabs-items>
      </v-card-text>
    </v-card>
  </div>
</template>

<script>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import CiTypeSchemas from './CiTypeSchemas.vue'
import RelationshipTypeSchemas from './RelationshipTypeSchemas.vue'

export default {
  name: 'SchemaList',
  components: {
    CiTypeSchemas,
    RelationshipTypeSchemas
  },
  setup() {
    const router = useRouter()
    const activeTab = ref(0)
    const ciTypeSchemas = ref(null)
    const relationshipTypeSchemas = ref(null)

    const handleTabChange = () => {
      // Refresh data when tab changes
      if (activeTab.value === 0 && ciTypeSchemas.value) {
        ciTypeSchemas.value.refresh()
      } else if (activeTab.value === 1 && relationshipTypeSchemas.value) {
        relationshipTypeSchemas.value.refresh()
      }
    }

    const navigateToCreate = () => {
      if (activeTab.value === 0) {
        router.push('/schemas/ci-types/create')
      } else {
        router.push('/schemas/relationship-types/create')
      }
    }

    return {
      activeTab,
      handleTabChange,
      navigateToCreate,
      ciTypeSchemas,
      relationshipTypeSchemas
    }
  }
}
</script>

<style scoped>
.schema-list {
  padding: 20px;
}
</style>
