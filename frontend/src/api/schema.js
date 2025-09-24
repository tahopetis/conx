import api from './index'

/**
 * Schema API Service
 * Handles all schema-related API calls
 */

export default {
  // CI Type Schema APIs
  async fetchCiTypeSchemas(params = {}) {
    const response = await api.get('/schemas/ci-types', { params })
    return response.data
  },

  async fetchCiTypeSchema(id) {
    const response = await api.get(`/schemas/ci-types/${id}`)
    return response.data
  },

  async createCiTypeSchema(schemaData) {
    const response = await api.post('/schemas/ci-types', schemaData)
    return response.data
  },

  async updateCiTypeSchema(id, schemaData) {
    const response = await api.put(`/schemas/ci-types/${id}`, schemaData)
    return response.data
  },

  async deleteCiTypeSchema(id) {
    const response = await api.delete(`/schemas/ci-types/${id}`)
    return response.data
  },

  // Relationship Type Schema APIs
  async fetchRelationshipTypeSchemas(params = {}) {
    const response = await api.get('/schemas/relationship-types', { params })
    return response.data
  },

  async fetchRelationshipTypeSchema(id) {
    const response = await api.get(`/schemas/relationship-types/${id}`)
    return response.data
  },

  async createRelationshipTypeSchema(schemaData) {
    const response = await api.post('/schemas/relationship-types', schemaData)
    return response.data
  },

  async updateRelationshipTypeSchema(id, schemaData) {
    const response = await api.put(`/schemas/relationship-types/${id}`, schemaData)
    return response.data
  },

  async deleteRelationshipTypeSchema(id) {
    const response = await api.delete(`/schemas/relationship-types/${id}`)
    return response.data
  },

  // Schema Template APIs
  async fetchCiTypeTemplates() {
    const response = await api.get('/schemas/templates/ci')
    return response.data
  },

  async fetchRelationshipTypeTemplates() {
    const response = await api.get('/schemas/templates/relationship')
    return response.data
  },

  async createSchemaFromTemplate(templateName, schemaType = 'ci_type') {
    const endpoint = schemaType === 'ci_type' 
      ? `/schemas/templates/ci/${templateName}`
      : `/schemas/templates/relationship/${templateName}`
    
    const response = await api.post(endpoint)
    return response.data
  },

  // Schema Validation APIs
  async validateCIAgainstSchema(ciData, schemaData) {
    const response = await api.post('/schemas/validate/ci', {
      ci: ciData,
      schema: schemaData
    })
    return response.data
  },

  async validateRelationshipAgainstSchema(relationshipData, schemaData) {
    const response = await api.post('/schemas/validate/relationship', {
      relationship: relationshipData,
      schema: schemaData
    })
    return response.data
  }
}
