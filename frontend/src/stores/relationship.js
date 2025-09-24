import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import api from '@/services/api'

export const useRelationshipStore = defineStore('relationship', () => {
  // State
  const relationships = ref([])
  const relationshipsTotal = ref(0)
  const relationshipsLoading = ref(false)
  const currentRelationship = ref(null)
  const relationshipCache = ref(new Map())
  
  const relationshipGraph = ref(null)
  const graphLoading = ref(false)
  
  const filters = ref({
    type: '',
    source_type: '',
    target_type: '',
    search: ''
  })

  // Computed
  const relationshipOptions = computed(() => 
    relationships.value
      .filter(rel => rel.is_active)
      .map(rel => ({
        text: `${rel.source_name} -> ${rel.target_name}`,
        value: rel.id,
        description: rel.type
      }))
  )
  
  const filteredRelationships = computed(() => {
    let result = relationships.value
    
    if (filters.value.type) {
      result = result.filter(rel => rel.type === filters.value.type)
    }
    
    if (filters.value.source_type) {
      result = result.filter(rel => rel.source_type === filters.value.source_type)
    }
    
    if (filters.value.target_type) {
      result = result.filter(rel => rel.target_type === filters.value.target_type)
    }
    
    if (filters.value.search) {
      const searchTerm = filters.value.search.toLowerCase()
      result = result.filter(rel => 
        rel.source_name.toLowerCase().includes(searchTerm) ||
        rel.target_name.toLowerCase().includes(searchTerm) ||
        rel.type.toLowerCase().includes(searchTerm) ||
        (rel.attributes && Object.values(rel.attributes).some(attr => 
          String(attr).toLowerCase().includes(searchTerm)
        ))
      )
    }
    
    return result
  })

  // Actions
  const setRelationships = ({ items, total }) => {
    relationships.value = items
    relationshipsTotal.value = total
  }
  
  const setRelationshipsLoading = (loading) => {
    relationshipsLoading.value = loading
  }
  
  const setCurrentRelationship = (relationship) => {
    currentRelationship.value = relationship
    if (relationship) {
      relationshipCache.value.set(relationship.id, relationship)
    }
  }
  
  const setRelationshipGraph = (graph) => {
    relationshipGraph.value = graph
  }
  
  const setGraphLoading = (loading) => {
    graphLoading.value = loading
  }
  
  const setFilters = (newFilters) => {
    filters.value = { ...filters.value, ...newFilters }
  }
  
  const updateRelationshipInCache = (relationship) => {
    if (relationship && relationship.id) {
      relationshipCache.value.set(relationship.id, relationship)
    }
  }
  
  const removeRelationshipFromCache = (id) => {
    relationshipCache.value.delete(id)
  }

  // CRUD Actions
  const fetchRelationships = async (params = {}) => {
    setRelationshipsLoading(true)
    try {
      const response = await api.get('/api/v1/relationships', { params })
      setRelationships({
        items: response.data.items,
        total: response.data.total_count
      })
      return response.data
    } finally {
      setRelationshipsLoading(false)
    }
  }
  
  const fetchRelationship = async (id) => {
    // Check cache first
    if (relationshipCache.value.has(id)) {
      setCurrentRelationship(relationshipCache.value.get(id))
      return relationshipCache.value.get(id)
    }
    
    try {
      const response = await api.get(`/api/v1/relationships/${id}`)
      const relationship = response.data
      setCurrentRelationship(relationship)
      return relationship
    } catch (error) {
      console.error('Failed to fetch relationship:', error)
      throw error
    }
  }
  
  const createRelationship = async (relationshipData) => {
    try {
      const response = await api.post('/api/v1/relationships', relationshipData)
      const relationship = response.data
      setCurrentRelationship(relationship)
      return relationship
    } catch (error) {
      console.error('Failed to create relationship:', error)
      throw error
    }
  }
  
  const updateRelationship = async ({ id, ...relationshipData }) => {
    try {
      const response = await api.put(`/api/v1/relationships/${id}`, relationshipData)
      const relationship = response.data
      setCurrentRelationship(relationship)
      updateRelationshipInCache(relationship)
      return relationship
    } catch (error) {
      console.error('Failed to update relationship:', error)
      throw error
    }
  }
  
  const deleteRelationship = async (id) => {
    try {
      await api.delete(`/api/v1/relationships/${id}`)
      // Remove from cache
      removeRelationshipFromCache(id)
      // Clear current if it's the deleted one
      if (currentRelationship.value?.id === id) {
        setCurrentRelationship(null)
      }
      // Remove from list
      relationships.value = relationships.value.filter(rel => rel.id !== id)
    } catch (error) {
      console.error('Failed to delete relationship:', error)
      throw error
    }
  }
  
  // Graph Actions
  const fetchRelationshipGraph = async (params = {}) => {
    setGraphLoading(true)
    try {
      const response = await api.get('/api/v1/relationships/graph', { params })
      setRelationshipGraph(response.data)
      return response.data
    } finally {
      setGraphLoading(false)
    }
  }
  
  // Filter Actions
  const applyFilters = (newFilters) => {
    setFilters(newFilters)
  }
  
  const clearFilters = () => {
    setFilters({
      type: '',
      source_type: '',
      target_type: '',
      search: ''
    })
  }

  // Utility Getters
  const getRelationshipById = (id) => relationshipCache.value.get(id)
  const getRelationshipsByType = (type) => relationships.value.filter(rel => rel.type === type)
  const getRelationshipsBySource = (sourceId) => relationships.value.filter(rel => rel.source_id === sourceId)
  const getRelationshipsByTarget = (targetId) => relationships.value.filter(rel => rel.target_id === targetId)
  const getRelationshipsByCI = (ciId) => relationships.value.filter(rel => rel.source_id === ciId || rel.target_id === ciId)

  return {
    // State
    relationships,
    relationshipsTotal,
    relationshipsLoading,
    currentRelationship,
    relationshipCache,
    relationshipGraph,
    graphLoading,
    filters,
    
    // Computed
    relationshipOptions,
    filteredRelationships,
    
    // Actions
    setRelationships,
    setRelationshipsLoading,
    setCurrentRelationship,
    setRelationshipGraph,
    setGraphLoading,
    setFilters,
    updateRelationshipInCache,
    removeRelationshipFromCache,
    fetchRelationships,
    fetchRelationship,
    createRelationship,
    updateRelationship,
    deleteRelationship,
    fetchRelationshipGraph,
    applyFilters,
    clearFilters,
    
    // Utility Getters
    getRelationshipById,
    getRelationshipsByType,
    getRelationshipsBySource,
    getRelationshipsByTarget,
    getRelationshipsByCI
  }
})
