<template>
  <div class="schema-create">
    <v-card>
      <v-card-title class="d-flex justify-space-between align-center">
        <span>Create New Schema</span>
        <v-btn icon @click="$router.go(-1)">
          <v-icon>mdi-close</v-icon>
        </v-btn>
      </v-card-title>

      <v-card-text>
        <v-form ref="form" v-model="valid" @submit.prevent="handleSubmit">
          <!-- Schema Type Selection -->
          <v-row>
            <v-col cols="12">
              <v-radio-group v-model="schemaType" row>
                <v-radio label="CI Type Schema" value="ci_type"></v-radio>
                <v-radio label="Relationship Type Schema" value="relationship_type"></v-radio>
              </v-radio-group>
            </v-col>
          </v-row>

          <!-- Basic Information -->
          <v-row>
            <v-col cols="12" md="6">
              <v-text-field
                v-model="schema.name"
                label="Schema Name *"
                :rules="[rules.required, rules.nameFormat]"
                required
              />
            </v-col>
            <v-col cols="12" md="6">
              <v-text-field
                v-model="schema.description"
                label="Description"
                :rules="[rules.maxLength(500)]"
              />
            </v-col>
          </v-row>

          <!-- Attributes Section -->
          <v-row>
            <v-col cols="12">
              <div class="d-flex justify-space-between align-center mb-4">
                <h3>Attributes</h3>
                <v-btn color="primary" small @click="addAttribute">
                  <v-icon left>mdi-plus</v-icon>
                  Add Attribute
                </v-btn>
              </div>

              <!-- Attributes List -->
              <div v-for="(attribute, index) in schema.attributes" :key="index" class="attribute-item mb-4">
                <v-card outlined>
                  <v-card-title class="d-flex justify-space-between align-center py-2">
                    <span class="text-subtitle-2">Attribute {{ index + 1 }}</span>
                    <v-btn icon small color="error" @click="removeAttribute(index)">
                      <v-icon small>mdi-delete</v-icon>
                    </v-btn>
                  </v-card-title>
                  <v-card-text>
                    <v-row>
                      <v-col cols="12" md="4">
                        <v-text-field
                          v-model="attribute.name"
                          label="Attribute Name *"
                          :rules="[rules.required, rules.attributeName]"
                          required
                          @input="validateAttributeName(attribute, index)"
                        />
                      </v-col>
                      <v-col cols="12" md="4">
                        <v-select
                          v-model="attribute.type"
                          label="Data Type *"
                          :items="dataTypes"
                          :rules="[rules.required]"
                          required
                          @change="handleTypeChange(attribute)"
                        />
                      </v-col>
                      <v-col cols="12" md="4">
                        <v-switch
                          v-model="attribute.required"
                          label="Required"
                          inset
                        />
                      </v-col>
                    </v-row>
                    <v-row>
                      <v-col cols="12" md="6">
                        <v-text-field
                          v-model="attribute.description"
                          label="Description"
                          :rules="[rules.maxLength(200)]"
                        />
                      </v-col>
                      <v-col cols="12" md="6">
                        <v-text-field
                          v-model="attribute.default"
                          label="Default Value"
                          :placeholder="getPlaceholderForType(attribute.type)"
                          @input="validateDefaultValue(attribute)"
                        />
                      </v-col>
                    </v-row>

                    <!-- Validation Rules -->
                    <div class="validation-rules mt-4">
                      <h4 class="text-subtitle-2 mb-2">Validation Rules</h4>
                      <v-row>
                        <v-col cols="12" md="6">
                          <v-text-field
                            v-if="supportsValidation(attribute.type, 'min')"
                            v-model.number="attribute.validation.min"
                            label="Minimum Value"
                            type="number"
                            @input="validateValidationRules(attribute)"
                          />
                        </v-col>
                        <v-col cols="12" md="6">
                          <v-text-field
                            v-if="supportsValidation(attribute.type, 'max')"
                            v-model.number="attribute.validation.max"
                            label="Maximum Value"
                            type="number"
                            @input="validateValidationRules(attribute)"
                          />
                        </v-col>
                        <v-col cols="12" md="6">
                          <v-text-field
                            v-if="supportsValidation(attribute.type, 'minLength')"
                            v-model.number="attribute.validation.minLength"
                            label="Minimum Length"
                            type="number"
                            @input="validateValidationRules(attribute)"
                          />
                        </v-col>
                        <v-col cols="12" md="6">
                          <v-text-field
                            v-if="supportsValidation(attribute.type, 'maxLength')"
                            v-model.number="attribute.validation.maxLength"
                            label="Maximum Length"
                            type="number"
                            @input="validateValidationRules(attribute)"
                          />
                        </v-col>
                        <v-col cols="12" md="6">
                          <v-text-field
                            v-if="supportsValidation(attribute.type, 'pattern')"
                            v-model="attribute.validation.pattern"
                            label="Pattern (Regex)"
                            placeholder="e.g., ^[a-zA-Z0-9]+$"
                            @input="validateValidationRules(attribute)"
                          />
                        </v-col>
                        <v-col cols="12" md="6">
                          <v-select
                            v-if="supportsValidation(attribute.type, 'format')"
                            v-model="attribute.validation.format"
                            label="Format"
                            :items="formatOptions"
                            @change="validateValidationRules(attribute)"
                          />
                        </v-col>
                        <v-col cols="12" md="6">
                          <v-combobox
                            v-if="supportsValidation(attribute.type, 'enum')"
                            v-model="attribute.validation.enum"
                            label="Enum Values"
                            multiple
                            chips
                            small-chips
                            deletable-chips
                            placeholder="Enter allowed values"
                            @input="validateValidationRules(attribute)"
                          />
                        </v-col>
                      </v-row>
                    </div>
                  </v-card-text>
                </v-card>
              </div>

              <!-- Empty State -->
              <div v-if="schema.attributes.length === 0" class="text-center py-8">
                <v-icon large color="grey">mdi-information-outline</v-icon>
                <p class="text-grey mt-2">No attributes defined. Click "Add Attribute" to get started.</p>
              </div>
            </v-col>
          </v-row>

          <!-- Form Actions -->
          <v-row class="mt-6">
            <v-col cols="12" class="text-right">
              <v-btn text @click="$router.go(-1)" class="mr-2">Cancel</v-btn>
              <v-btn
                color="primary"
                type="submit"
                :loading="loading"
                :disabled="!valid || schema.attributes.length === 0"
              >
                Create Schema
              </v-btn>
            </v-col>
          </v-row>
        </v-form>
      </v-card-text>
    </v-card>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { useSchemaStore } from '@/stores/schema'

const router = useRouter()
const schemaStore = useSchemaStore()
const form = ref(null)
const valid = ref(false)
const loading = ref(false)
const schemaType = ref('ci_type')

// Schema data
const schema = reactive({
  name: '',
  description: '',
  attributes: []
})

// Data types
const dataTypes = [
  { text: 'String', value: 'string' },
  { text: 'Number', value: 'number' },
  { text: 'Boolean', value: 'boolean' },
  { text: 'Date', value: 'date' },
  { text: 'Array', value: 'array' },
  { text: 'Object', value: 'object' }
]

// Format options
const formatOptions = [
  { text: 'Email', value: 'email' },
  { text: 'IPv4', value: 'ipv4' },
  { text: 'IPv6', value: 'ipv6' },
  { text: 'URL', value: 'url' },
  { text: 'UUID', value: 'uuid' }
]

// Validation rules
const rules = {
  required: (v) => !!v || 'This field is required',
  nameFormat: (v) => /^[a-z_][a-z0-9_]*$/.test(v) || 'Name must be lowercase with underscores',
  attributeName: (v) => /^[a-z_][a-z0-9_]*$/.test(v) || 'Attribute name must be lowercase with underscores',
  maxLength: (max) => (v) => !v || v.length <= max || `Maximum ${max} characters allowed`
}

// Methods
const addAttribute = () => {
  schema.attributes.push({
    name: '',
    type: 'string',
    required: false,
    description: '',
    default: '',
    validation: {}
  })
}

const removeAttribute = (index) => {
  schema.attributes.splice(index, 1)
}

const handleTypeChange = (attribute) => {
  // Reset validation rules when type changes
  attribute.validation = {}
  attribute.default = ''
}

const validateAttributeName = (attribute, index) => {
  // Check for duplicate attribute names
  const duplicateIndex = schema.attributes.findIndex((attr, i) => 
    i !== index && attr.name === attribute.name
  )
  
  if (duplicateIndex !== -1) {
    console.warn('Attribute name must be unique')
  }
}

const validateDefaultValue = (attribute) => {
  // Validate default value against type and validation rules
  if (attribute.default) {
    try {
      // Basic type validation
      switch (attribute.type) {
        case 'number':
          if (isNaN(attribute.default)) {
            console.warn('Default value must be a number')
            attribute.default = ''
          }
          break
        case 'boolean':
          if (attribute.default !== 'true' && attribute.default !== 'false') {
            console.warn('Default value must be true or false')
            attribute.default = ''
          }
          break
      }
    } catch (error) {
      console.error('Invalid default value')
      attribute.default = ''
    }
  }
}

const validateValidationRules = (attribute) => {
  // Validate validation rules consistency
  const { validation } = attribute
  
  if (validation.min !== undefined && validation.max !== undefined) {
    if (validation.min > validation.max) {
      console.warn('Minimum value cannot be greater than maximum value')
    }
  }
  
  if (validation.minLength !== undefined && validation.maxLength !== undefined) {
    if (validation.minLength > validation.maxLength) {
      console.warn('Minimum length cannot be greater than maximum length')
    }
  }
}

const supportsValidation = (type, validationType) => {
  const validationMap = {
    string: ['minLength', 'maxLength', 'pattern', 'format', 'enum'],
    number: ['min', 'max'],
    date: ['min', 'max'],
    array: ['minLength', 'maxLength'],
    object: [],
    boolean: []
  }
  
  return validationMap[type]?.includes(validationType) || false
}

const getPlaceholderForType = (type) => {
  const placeholders = {
    string: 'Enter text value',
    number: 'Enter number',
    boolean: 'true or false',
    date: 'YYYY-MM-DD',
    array: 'Enter array values',
    object: 'Enter JSON object'
  }
  return placeholders[type] || 'Enter value'
}

const validateSchema = () => {
  // Validate overall schema
  if (!schema.name) {
    console.warn('Schema name is required')
    return false
  }

  if (schema.attributes.length === 0) {
    console.warn('At least one attribute is required')
    return false
  }

  // Validate each attribute
  for (let i = 0; i < schema.attributes.length; i++) {
    const attribute = schema.attributes[i]
    
    if (!attribute.name) {
      console.warn(`Attribute ${i + 1} name is required`)
      return false
    }

    if (!attribute.type) {
      console.warn(`Attribute ${i + 1} type is required`)
      return false
    }

    // Check for duplicate names
    const duplicateIndex = schema.attributes.findIndex((attr, index) => 
      index !== i && attr.name === attribute.name
    )
    
    if (duplicateIndex !== -1) {
      console.warn(`Duplicate attribute name: ${attribute.name}`)
      return false
    }
  }

  return true
}

const handleSubmit = async () => {
  if (!form.value.validate() || !validateSchema()) {
    return
  }

  loading.value = true

  try {
    // Prepare schema data
    const schemaData = {
      name: schema.name,
      description: schema.description,
      attributes: schema.attributes.map(attr => ({
        ...attr,
        validation: attr.validation || {}
      })),
      is_active: true
    }

    // Create schema based on type
    let createdSchema
    if (schemaType.value === 'ci_type') {
      createdSchema = await schemaStore.createCiTypeSchema(schemaData)
    } else {
      createdSchema = await schemaStore.createRelationshipTypeSchema(schemaData)
    }
    
    // Navigate to schema details
    if (schemaType.value === 'ci_type') {
      router.push(`/schemas/ci-types/${createdSchema.id}`)
    } else {
      router.push(`/schemas/relationship-types/${createdSchema.id}`)
    }
  } catch (error) {
    console.error('Failed to create schema:', error)
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.schema-create {
  padding: 20px;
}

.attribute-item {
  border-left: 4px solid #1976d2;
}

.validation-rules {
  background-color: #f5f5f5;
  padding: 16px;
  border-radius: 4px;
}
</style>
