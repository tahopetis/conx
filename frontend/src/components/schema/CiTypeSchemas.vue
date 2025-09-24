<template>
  <div class="ci-type-schemas">
    <!-- Search and Filter -->
    <v-row class="mb-4">
      <v-col cols="12" md="6">
        <v-text-field
          v-model="searchQuery"
          label="Search schemas..."
          prepend-inner-icon="mdi-magnify"
          clearable
          @input="handleSearch"
        />
      </v-col>
      <v-col cols="12" md="6" class="text-right">
        <v-btn
          color="primary"
          outlined
          @click="showTemplates = true"
        >
          <v-icon left>mdi-template-multiple</v-icon>
          Use Template
        </v-btn>
      </v-col>
    </v-row>

    <!-- Schemas Table -->
    <v-data-table
      :headers="headers"
      :items="schemas"
      :loading="loading"
      :options.sync="options"
      :server-items-length="totalSchemas"
      :footer-props="{
        'items-per-page-options': [10, 20, 50, 100]
      }"
      class="elevation-1"
    >
      <template v-slot:item.name="{ item }">
        <div class="d-flex align-center">
          <v-icon left :color="item.is_active ? 'success' : 'grey'">
            {{ item.is_active ? 'mdi-check-circle' : 'mdi-circle-outline' }}
          </v-icon>
          <span>{{ item.name }}</span>
        </div>
      </template>

      <template v-slot:item.attributes="{ item }">
        <v-chip small color="info">
          {{ item.attributes ? item.attributes.length : 0 }} attributes
        </v-chip>
      </template>

      <template v-slot:item.is_active="{ item }">
        <v-chip :color="item.is_active ? 'success' : 'grey'" small>
          {{ item.is_active ? 'Active' : 'Inactive' }}
        </v-chip>
      </template>

      <template v-slot:item.created_at="{ item }">
        {{ formatDate(item.created_at) }}
      </template>

      <template v-slot:item.actions="{ item }">
        <div class="d-flex justify-end">
          <v-btn
            icon
            small
            color="primary"
            @click="viewSchema(item)"
            title="View Details"
          >
            <v-icon small>mdi-eye</v-icon>
          </v-btn>
          <v-btn
            icon
            small
            color="warning"
            @click="editSchema(item)"
            title="Edit Schema"
          >
            <v-icon small>mdi-pencil</v-icon>
          </v-btn>
          <v-btn
            icon
            small
            :color="item.is_active ? 'error' : 'success'"
            @click="toggleSchemaStatus(item)"
            :title="item.is_active ? 'Deactivate' : 'Activate'"
          >
            <v-icon small>{{ item.is_active ? 'mdi-delete' : 'mdi-restore' }}</v-icon>
          </v-btn>
        </div>
      </template>
    </v-data-table>

    <!-- Templates Dialog -->
    <v-dialog v-model="showTemplates" max-width="800px">
      <v-card>
        <v-card-title class="d-flex justify-space-between align-center">
          <span>Schema Templates</span>
          <v-btn icon @click="showTemplates = false">
            <v-icon>mdi-close</v-icon>
          </v-btn>
        </v-card-title>
        <v-card-text>
          <v-row>
            <v-col
              v-for="template in templates"
              :key="template.name"
              cols="12"
              md="6"
            >
              <v-card
                outlined
                hover
                @click="createFromTemplate(template)"
                class="template-card"
              >
                <v-card-title class="text-subtitle-2">
                  <v-icon left color="primary">mdi-file-document-outline</v-icon>
                  {{ template.name }}
                </v-card-title>
                <v-card-text>
                  <p class="text-caption mb-0">{{ template.description }}</p>
                  <div class="mt-2">
                    <v-chip x-small color="info" class="mr-1">
                      {{ template.attributes ? template.attributes.length : 0 }} attributes
                    </v-chip>
                  </div>
                </v-card-text>
              </v-card>
            </v-col>
          </v-row>
        </v-card-text>
      </v-card>
    </v-dialog>

    <!-- Delete Confirmation Dialog -->
    <v-dialog v-model="showDeleteDialog" max-width="400px">
      <v-card>
        <v-card-title>Confirm Status Change</v-card-title>
        <v-card-text>
          Are you sure you want to {{ schemaToDelete?.is_active ? 'deactivate' : 'activate' }} 
          the schema "{{ schemaToDelete?.name }}"?
        </v-card-text>
        <v-card-actions>
          <v-spacer></v-spacer>
          <v-btn text @click="showDeleteDialog = false">Cancel</v-btn>
          <v-btn
            color="primary"
            @click="confirmStatusToggle"
          >
            {{ schemaToDelete?.is_active ? 'Deactivate' : 'Activate' }}
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useSchemaStore } from '@/stores/schema'
import { debounce } from 'lodash-es'

const router = useRouter()
const schemaStore = useSchemaStore()

// Data
const schemas = ref([])
const templates = ref([])
const loading = ref(false)
const totalSchemas = ref(0)
const searchQuery = ref('')
const showTemplates = ref(false)
const showDeleteDialog = ref(false)
const schemaToDelete = ref(null)

// Table options
const options = ref({
  page: 1,
  itemsPerPage: 20,
  sortBy: ['name'],
  sortDesc: [false]
})

// Table headers
const headers = [
  { text: 'Name', value: 'name', sortable: true },
  { text: 'Description', value: 'description', sortable: false },
  { text: 'Attributes', value: 'attributes', sortable: false },
  { text: 'Status', value: 'is_active', sortable: true },
  { text: 'Created', value: 'created_at', sortable: true },
  { text: 'Actions', value: 'actions', sortable: false, align: 'end' }
]

// Methods
const loadSchemas = async () => {
  loading.value = true
  try {
    const params = {
      page: options.value.page,
      page_size: options.value.itemsPerPage,
      search: searchQuery.value,
      sort_by: options.value.sortBy[0],
      sort_order: options.value.sortDesc[0] ? 'desc' : 'asc'
    }

    const response = await schemaStore.fetchCiTypeSchemas(params)
    schemas.value = response.schemas
    totalSchemas.value = response.total_count
  } catch (error) {
    console.error('Failed to load schemas:', error)
  } finally {
    loading.value = false
  }
}

const loadTemplates = async () => {
  try {
    templates.value = await schemaStore.fetchCiTypeTemplates()
  } catch (error) {
    console.error('Failed to load templates:', error)
  }
}

const handleSearch = debounce(() => {
  options.value.page = 1
  loadSchemas()
}, 300)

const viewSchema = (schema) => {
  router.push(`/schemas/ci-types/${schema.id}`)
}

const editSchema = (schema) => {
  router.push(`/schemas/ci-types/${schema.id}/edit`)
}

const toggleSchemaStatus = (schema) => {
  schemaToDelete.value = schema
  showDeleteDialog.value = true
}

const confirmStatusToggle = async () => {
  try {
    await schemaStore.updateCiTypeSchema({
      id: schemaToDelete.value.id,
      is_active: !schemaToDelete.value.is_active
    })
    loadSchemas()
  } catch (error) {
    console.error('Failed to update schema status:', error)
  } finally {
    showDeleteDialog.value = false
    schemaToDelete.value = null
  }
}

const createFromTemplate = async (template) => {
  try {
    const newSchema = await schemaStore.createSchemaFromTemplate({
      templateName: template.name
    })
    showTemplates.value = false
    router.push(`/schemas/ci-types/${newSchema.id}/edit`)
  } catch (error) {
    console.error('Failed to create schema from template:', error)
  }
}

const formatDate = (dateString) => {
  return new Date(dateString).toLocaleDateString()
}

const refresh = () => {
  loadSchemas()
}

// Watch for options changes
watch(options, () => {
  loadSchemas()
}, { deep: true })

// Lifecycle
onMounted(() => {
  loadSchemas()
  loadTemplates()
})
</script>

<style scoped>
.ci-type-schemas {
  margin-top: 20px;
}

.template-card {
  cursor: pointer;
  transition: transform 0.2s;
}

.template-card:hover {
  transform: translateY(-2px);
}
</style>
