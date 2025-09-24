import schemaApi from '@/api/schema'

const state = {
  // CI Type Schemas
  ciTypeSchemas: [],
  ciTypeSchemasTotal: 0,
  ciTypeSchemasLoading: false,
  currentCiTypeSchema: null,
  
  // Relationship Type Schemas
  relationshipTypeSchemas: [],
  relationshipTypeSchemasTotal: 0,
  relationshipTypeSchemasLoading: false,
  currentRelationshipTypeSchema: null,
  
  // Templates
  ciTypeTemplates: [],
  relationshipTypeTemplates: [],
  
  // Validation Results
  validationResults: null,
  
  // Schema Cache
  schemaCache: new Map()
}

const mutations = {
  // CI Type Schema Mutations
  SET_CI_TYPE_SCHEMAS(state, { schemas, total }) {
    state.ciTypeSchemas = schemas
    state.ciTypeSchemasTotal = total
  },
  
  SET_CI_TYPE_SCHEMAS_LOADING(state, loading) {
    state.ciTypeSchemasLoading = loading
  },
  
  SET_CURRENT_CI_TYPE_SCHEMA(state, schema) {
    state.currentCiTypeSchema = schema
    if (schema) {
      state.schemaCache.set(schema.id, schema)
    }
  },
  
  // Relationship Type Schema Mutations
  SET_RELATIONSHIP_TYPE_SCHEMAS(state, { schemas, total }) {
    state.relationshipTypeSchemas = schemas
    state.relationshipTypeSchemasTotal = total
  },
  
  SET_RELATIONSHIP_TYPE_SCHEMAS_LOADING(state, loading) {
    state.relationshipTypeSchemasLoading = loading
  },
  
  SET_CURRENT_RELATIONSHIP_TYPE_SCHEMA(state, schema) {
    state.currentRelationshipTypeSchema = schema
    if (schema) {
      state.schemaCache.set(schema.id, schema)
    }
  },
  
  // Template Mutations
  SET_CI_TYPE_TEMPLATES(state, templates) {
    state.ciTypeTemplates = templates
  },
  
  SET_RELATIONSHIP_TYPE_TEMPLATES(state, templates) {
    state.relationshipTypeTemplates = templates
  },
  
  // Validation Mutations
  SET_VALIDATION_RESULTS(state, results) {
    state.validationResults = results
  },
  
  // Cache Mutations
  CLEAR_SCHEMA_CACHE(state) {
    state.schemaCache.clear()
  },
  
  UPDATE_SCHEMA_IN_CACHE(state, schema) {
    if (schema && schema.id) {
      state.schemaCache.set(schema.id, schema)
    }
  }
}

const actions = {
  // CI Type Schema Actions
  async fetchCiTypeSchemas({ commit }, params = {}) {
    commit('SET_CI_TYPE_SCHEMAS_LOADING', true)
    try {
      const response = await schemaApi.fetchCiTypeSchemas(params)
      commit('SET_CI_TYPE_SCHEMAS', {
        schemas: response.schemas,
        total: response.total_count
      })
      return response
    } finally {
      commit('SET_CI_TYPE_SCHEMAS_LOADING', false)
    }
  },
  
  async fetchCiTypeSchema({ commit, state }, id) {
    // Check cache first
    if (state.schemaCache.has(id)) {
      commit('SET_CURRENT_CI_TYPE_SCHEMA', state.schemaCache.get(id))
      return state.schemaCache.get(id)
    }
    
    try {
      const schema = await schemaApi.fetchCiTypeSchema(id)
      commit('SET_CURRENT_CI_TYPE_SCHEMA', schema)
      return schema
    } catch (error) {
      console.error('Failed to fetch CI type schema:', error)
      throw error
    }
  },
  
  async createCiTypeSchema({ commit }, schemaData) {
    try {
      const schema = await schemaApi.createCiTypeSchema(schemaData)
      commit('SET_CURRENT_CI_TYPE_SCHEMA', schema)
      return schema
    } catch (error) {
      console.error('Failed to create CI type schema:', error)
      throw error
    }
  },
  
  async updateCiTypeSchema({ commit }, { id, ...schemaData }) {
    try {
      const schema = await schemaApi.updateCiTypeSchema(id, schemaData)
      commit('SET_CURRENT_CI_TYPE_SCHEMA', schema)
      commit('UPDATE_SCHEMA_IN_CACHE', schema)
      return schema
    } catch (error) {
      console.error('Failed to update CI type schema:', error)
      throw error
    }
  },
  
  async deleteCiTypeSchema({ commit, state }, id) {
    try {
      await schemaApi.deleteCiTypeSchema(id)
      // Remove from cache
      state.schemaCache.delete(id)
      // Clear current if it's the deleted one
      if (state.currentCiTypeSchema?.id === id) {
        commit('SET_CURRENT_CI_TYPE_SCHEMA', null)
      }
    } catch (error) {
      console.error('Failed to delete CI type schema:', error)
      throw error
    }
  },
  
  // Relationship Type Schema Actions
  async fetchRelationshipTypeSchemas({ commit }, params = {}) {
    commit('SET_RELATIONSHIP_TYPE_SCHEMAS_LOADING', true)
    try {
      const response = await schemaApi.fetchRelationshipTypeSchemas(params)
      commit('SET_RELATIONSHIP_TYPE_SCHEMAS', {
        schemas: response.schemas,
        total: response.total_count
      })
      return response
    } finally {
      commit('SET_RELATIONSHIP_TYPE_SCHEMAS_LOADING', false)
    }
  },
  
  async fetchRelationshipTypeSchema({ commit, state }, id) {
    // Check cache first
    if (state.schemaCache.has(id)) {
      commit('SET_CURRENT_RELATIONSHIP_TYPE_SCHEMA', state.schemaCache.get(id))
      return state.schemaCache.get(id)
    }
    
    try {
      const schema = await schemaApi.fetchRelationshipTypeSchema(id)
      commit('SET_CURRENT_RELATIONSHIP_TYPE_SCHEMA', schema)
      return schema
    } catch (error) {
      console.error('Failed to fetch relationship type schema:', error)
      throw error
    }
  },
  
  async createRelationshipTypeSchema({ commit }, schemaData) {
    try {
      const schema = await schemaApi.createRelationshipTypeSchema(schemaData)
      commit('SET_CURRENT_RELATIONSHIP_TYPE_SCHEMA', schema)
      return schema
    } catch (error) {
      console.error('Failed to create relationship type schema:', error)
      throw error
    }
  },
  
  async updateRelationshipTypeSchema({ commit }, { id, ...schemaData }) {
    try {
      const schema = await schemaApi.updateRelationshipTypeSchema(id, schemaData)
      commit('SET_CURRENT_RELATIONSHIP_TYPE_SCHEMA', schema)
      commit('UPDATE_SCHEMA_IN_CACHE', schema)
      return schema
    } catch (error) {
      console.error('Failed to update relationship type schema:', error)
      throw error
    }
  },
  
  async deleteRelationshipTypeSchema({ commit, state }, id) {
    try {
      await schemaApi.deleteRelationshipTypeSchema(id)
      // Remove from cache
      state.schemaCache.delete(id)
      // Clear current if it's the deleted one
      if (state.currentRelationshipTypeSchema?.id === id) {
        commit('SET_CURRENT_RELATIONSHIP_TYPE_SCHEMA', null)
      }
    } catch (error) {
      console.error('Failed to delete relationship type schema:', error)
      throw error
    }
  },
  
  // Template Actions
  async fetchCiTypeTemplates({ commit }) {
    try {
      const templates = await schemaApi.fetchCiTypeTemplates()
      commit('SET_CI_TYPE_TEMPLATES', templates)
      return templates
    } catch (error) {
      console.error('Failed to fetch CI type templates:', error)
      throw error
    }
  },
  
  async fetchRelationshipTypeTemplates({ commit }) {
    try {
      const templates = await schemaApi.fetchRelationshipTypeTemplates()
      commit('SET_RELATIONSHIP_TYPE_TEMPLATES', templates)
      return templates
    } catch (error) {
      console.error('Failed to fetch relationship type templates:', error)
      throw error
    }
  },
  
  async createSchemaFromTemplate({ commit }, { templateName, schemaType = 'ci_type' }) {
    try {
      const schema = await schemaApi.createSchemaFromTemplate(templateName, schemaType)
      
      if (schemaType === 'ci_type') {
        commit('SET_CURRENT_CI_TYPE_SCHEMA', schema)
      } else {
        commit('SET_CURRENT_RELATIONSHIP_TYPE_SCHEMA', schema)
      }
      
      return schema
    } catch (error) {
      console.error('Failed to create schema from template:', error)
      throw error
    }
  },
  
  // Validation Actions
  async validateCIAgainstSchema({ commit }, { ciData, schemaData }) {
    try {
      const results = await schemaApi.validateCIAgainstSchema(ciData, schemaData)
      commit('SET_VALIDATION_RESULTS', results)
      return results
    } catch (error) {
      console.error('Failed to validate CI against schema:', error)
      throw error
    }
  },
  
  async validateRelationshipAgainstSchema({ commit }, { relationshipData, schemaData }) {
    try {
      const results = await schemaApi.validateRelationshipAgainstSchema(relationshipData, schemaData)
      commit('SET_VALIDATION_RESULTS', results)
      return results
    } catch (error) {
      console.error('Failed to validate relationship against schema:', error)
      throw error
    }
  },
  
  // Cache Actions
  clearSchemaCache({ commit }) {
    commit('CLEAR_SCHEMA_CACHE')
  }
}

const getters = {
  // CI Type Schema Getters
  ciTypeSchemas: state => state.ciTypeSchemas,
  ciTypeSchemasTotal: state => state.ciTypeSchemasTotal,
  ciTypeSchemasLoading: state => state.ciTypeSchemasLoading,
  currentCiTypeSchema: state => state.currentCiTypeSchema,
  
  // Relationship Type Schema Getters
  relationshipTypeSchemas: state => state.relationshipTypeSchemas,
  relationshipTypeSchemasTotal: state => state.relationshipTypeSchemasTotal,
  relationshipTypeSchemasLoading: state => state.relationshipTypeSchemasLoading,
  currentRelationshipTypeSchema: state => state.currentRelationshipTypeSchema,
  
  // Template Getters
  ciTypeTemplates: state => state.ciTypeTemplates,
  relationshipTypeTemplates: state => state.relationshipTypeTemplates,
  
  // Validation Getters
  validationResults: state => state.validationResults,
  
  // Utility Getters
  getSchemaById: state => (id) => state.schemaCache.get(id),
  getCiTypeSchemaByName: state => (name) => 
    state.ciTypeSchemas.find(schema => schema.name === name),
  getRelationshipTypeSchemaByName: state => (name) => 
    state.relationshipTypeSchemas.find(schema => schema.name === name),
  
  // Schema Options Getters
  ciTypeSchemaOptions: state => 
    state.ciTypeSchemas
      .filter(schema => schema.is_active)
      .map(schema => ({
        text: schema.name,
        value: schema.name,
        description: schema.description
      })),
      
  relationshipTypeSchemaOptions: state => 
    state.relationshipTypeSchemas
      .filter(schema => schema.is_active)
      .map(schema => ({
        text: schema.name,
        value: schema.name,
        description: schema.description
      }))
}

export default {
  namespaced: true,
  state,
  mutations,
  actions,
  getters
}
