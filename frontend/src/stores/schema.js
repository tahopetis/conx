import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import schemaApi from '@/api/schema'

export const useSchemaStore = defineStore('schema', () => {
  // State
  const ciTypeSchemas = ref([])
  const ciTypeSchemasTotal = ref(0)
  const ciTypeSchemasLoading = ref(false)
  const currentCiTypeSchema = ref(null)
  
  const relationshipTypeSchemas = ref([])
  const relationshipTypeSchemasTotal = ref(0)
  const relationshipTypeSchemasLoading = ref(false)
  const currentRelationshipTypeSchema = ref(null)
  
  const ciTypeTemplates = ref([])
  const relationshipTypeTemplates = ref([])
  
  const validationResults = ref(null)
  
  const schemaCache = ref(new Map())

  // Computed (Getters)
  const ciTypeSchemaOptions = computed(() => 
    ciTypeSchemas.value
      .filter(schema => schema.is_active)
      .map(schema => ({
        text: schema.name,
        value: schema.name,
        description: schema.description
      }))
  )
  
  const relationshipTypeSchemaOptions = computed(() => 
    relationshipTypeSchemas.value
      .filter(schema => schema.is_active)
      .map(schema => ({
        text: schema.name,
        value: schema.name,
        description: schema.description
      }))
  )

  // Actions
  const setCiTypeSchemas = ({ schemas, total }) => {
    ciTypeSchemas.value = schemas
    ciTypeSchemasTotal.value = total
  }
  
  const setCiTypeSchemasLoading = (loading) => {
    ciTypeSchemasLoading.value = loading
  }
  
  const setCurrentCiTypeSchema = (schema) => {
    currentCiTypeSchema.value = schema
    if (schema) {
      schemaCache.value.set(schema.id, schema)
    }
  }
  
  const setRelationshipTypeSchemas = ({ schemas, total }) => {
    relationshipTypeSchemas.value = schemas
    relationshipTypeSchemasTotal.value = total
  }
  
  const setRelationshipTypeSchemasLoading = (loading) => {
    relationshipTypeSchemasLoading.value = loading
  }
  
  const setCurrentRelationshipTypeSchema = (schema) => {
    currentRelationshipTypeSchema.value = schema
    if (schema) {
      schemaCache.value.set(schema.id, schema)
    }
  }
  
  const setCiTypeTemplates = (templates) => {
    ciTypeTemplates.value = templates
  }
  
  const setRelationshipTypeTemplates = (templates) => {
    relationshipTypeTemplates.value = templates
  }
  
  const setValidationResults = (results) => {
    validationResults.value = results
  }
  
  const clearSchemaCache = () => {
    schemaCache.value.clear()
  }
  
  const updateSchemaInCache = (schema) => {
    if (schema && schema.id) {
      schemaCache.value.set(schema.id, schema)
    }
  }

  // CI Type Schema Actions
  const fetchCiTypeSchemas = async (params = {}) => {
    setCiTypeSchemasLoading(true)
    try {
      const response = await schemaApi.fetchCiTypeSchemas(params)
      setCiTypeSchemas({
        schemas: response.schemas,
        total: response.total_count
      })
      return response
    } finally {
      setCiTypeSchemasLoading(false)
    }
  }
  
  const fetchCiTypeSchema = async (id) => {
    // Check cache first
    if (schemaCache.value.has(id)) {
      setCurrentCiTypeSchema(schemaCache.value.get(id))
      return schemaCache.value.get(id)
    }
    
    try {
      const schema = await schemaApi.fetchCiTypeSchema(id)
      setCurrentCiTypeSchema(schema)
      return schema
    } catch (error) {
      console.error('Failed to fetch CI type schema:', error)
      throw error
    }
  }
  
  const createCiTypeSchema = async (schemaData) => {
    try {
      const schema = await schemaApi.createCiTypeSchema(schemaData)
      setCurrentCiTypeSchema(schema)
      return schema
    } catch (error) {
      console.error('Failed to create CI type schema:', error)
      throw error
    }
  }
  
  const updateCiTypeSchema = async ({ id, ...schemaData }) => {
    try {
      const schema = await schemaApi.updateCiTypeSchema(id, schemaData)
      setCurrentCiTypeSchema(schema)
      updateSchemaInCache(schema)
      return schema
    } catch (error) {
      console.error('Failed to update CI type schema:', error)
      throw error
    }
  }
  
  const deleteCiTypeSchema = async (id) => {
    try {
      await schemaApi.deleteCiTypeSchema(id)
      // Remove from cache
      schemaCache.value.delete(id)
      // Clear current if it's the deleted one
      if (currentCiTypeSchema.value?.id === id) {
        setCurrentCiTypeSchema(null)
      }
    } catch (error) {
      console.error('Failed to delete CI type schema:', error)
      throw error
    }
  }
  
  // Relationship Type Schema Actions
  const fetchRelationshipTypeSchemas = async (params = {}) => {
    setRelationshipTypeSchemasLoading(true)
    try {
      const response = await schemaApi.fetchRelationshipTypeSchemas(params)
      setRelationshipTypeSchemas({
        schemas: response.schemas,
        total: response.total_count
      })
      return response
    } finally {
      setRelationshipTypeSchemasLoading(false)
    }
  }
  
  const fetchRelationshipTypeSchema = async (id) => {
    // Check cache first
    if (schemaCache.value.has(id)) {
      setCurrentRelationshipTypeSchema(schemaCache.value.get(id))
      return schemaCache.value.get(id)
    }
    
    try {
      const schema = await schemaApi.fetchRelationshipTypeSchema(id)
      setCurrentRelationshipTypeSchema(schema)
      return schema
    } catch (error) {
      console.error('Failed to fetch relationship type schema:', error)
      throw error
    }
  }
  
  const createRelationshipTypeSchema = async (schemaData) => {
    try {
      const schema = await schemaApi.createRelationshipTypeSchema(schemaData)
      setCurrentRelationshipTypeSchema(schema)
      return schema
    } catch (error) {
      console.error('Failed to create relationship type schema:', error)
      throw error
    }
  }
  
  const updateRelationshipTypeSchema = async ({ id, ...schemaData }) => {
    try {
      const schema = await schemaApi.updateRelationshipTypeSchema(id, schemaData)
      setCurrentRelationshipTypeSchema(schema)
      updateSchemaInCache(schema)
      return schema
    } catch (error) {
      console.error('Failed to update relationship type schema:', error)
      throw error
    }
  }
  
  const deleteRelationshipTypeSchema = async (id) => {
    try {
      await schemaApi.deleteRelationshipTypeSchema(id)
      // Remove from cache
      schemaCache.value.delete(id)
      // Clear current if it's the deleted one
      if (currentRelationshipTypeSchema.value?.id === id) {
        setCurrentRelationshipTypeSchema(null)
      }
    } catch (error) {
      console.error('Failed to delete relationship type schema:', error)
      throw error
    }
  }
  
  // Template Actions
  const fetchCiTypeTemplates = async () => {
    try {
      const templates = await schemaApi.fetchCiTypeTemplates()
      setCiTypeTemplates(templates)
      return templates
    } catch (error) {
      console.error('Failed to fetch CI type templates:', error)
      throw error
    }
  }
  
  const fetchRelationshipTypeTemplates = async () => {
    try {
      const templates = await schemaApi.fetchRelationshipTypeTemplates()
      setRelationshipTypeTemplates(templates)
      return templates
    } catch (error) {
      console.error('Failed to fetch relationship type templates:', error)
      throw error
    }
  }
  
  const createSchemaFromTemplate = async ({ templateName, schemaType = 'ci_type' }) => {
    try {
      const schema = await schemaApi.createSchemaFromTemplate(templateName, schemaType)
      
      if (schemaType === 'ci_type') {
        setCurrentCiTypeSchema(schema)
      } else {
        setCurrentRelationshipTypeSchema(schema)
      }
      
      return schema
    } catch (error) {
      console.error('Failed to create schema from template:', error)
      throw error
    }
  }
  
  // Validation Actions
  const validateCIAgainstSchema = async ({ ciData, schemaData }) => {
    try {
      const results = await schemaApi.validateCIAgainstSchema(ciData, schemaData)
      setValidationResults(results)
      return results
    } catch (error) {
      console.error('Failed to validate CI against schema:', error)
      throw error
    }
  }
  
  const validateRelationshipAgainstSchema = async ({ relationshipData, schemaData }) => {
    try {
      const results = await schemaApi.validateRelationshipAgainstSchema(relationshipData, schemaData)
      setValidationResults(results)
      return results
    } catch (error) {
      console.error('Failed to validate relationship against schema:', error)
      throw error
    }
  }

  // Utility Getters
  const getSchemaById = (id) => schemaCache.value.get(id)
  const getCiTypeSchemaByName = (name) => 
    ciTypeSchemas.value.find(schema => schema.name === name)
  const getRelationshipTypeSchemaByName = (name) => 
    relationshipTypeSchemas.value.find(schema => schema.name === name)

  return {
    // State
    ciTypeSchemas,
    ciTypeSchemasTotal,
    ciTypeSchemasLoading,
    currentCiTypeSchema,
    relationshipTypeSchemas,
    relationshipTypeSchemasTotal,
    relationshipTypeSchemasLoading,
    currentRelationshipTypeSchema,
    ciTypeTemplates,
    relationshipTypeTemplates,
    validationResults,
    schemaCache,
    
    // Computed
    ciTypeSchemaOptions,
    relationshipTypeSchemaOptions,
    
    // Actions
    setCiTypeSchemas,
    setCiTypeSchemasLoading,
    setCurrentCiTypeSchema,
    setRelationshipTypeSchemas,
    setRelationshipTypeSchemasLoading,
    setCurrentRelationshipTypeSchema,
    setCiTypeTemplates,
    setRelationshipTypeTemplates,
    setValidationResults,
    clearSchemaCache,
    updateSchemaInCache,
    fetchCiTypeSchemas,
    fetchCiTypeSchema,
    createCiTypeSchema,
    updateCiTypeSchema,
    deleteCiTypeSchema,
    fetchRelationshipTypeSchemas,
    fetchRelationshipTypeSchema,
    createRelationshipTypeSchema,
    updateRelationshipTypeSchema,
    deleteRelationshipTypeSchema,
    fetchCiTypeTemplates,
    fetchRelationshipTypeTemplates,
    createSchemaFromTemplate,
    validateCIAgainstSchema,
    validateRelationshipAgainstSchema,
    
    // Utility Getters
    getSchemaById,
    getCiTypeSchemaByName,
    getRelationshipTypeSchemaByName
  }
})
