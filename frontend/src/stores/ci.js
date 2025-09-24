import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import api from '@/services/api'

export const useCIStore = defineStore('ci', () => {
  // State
  const cis = ref([])
  const cisTotal = ref(0)
  const cisLoading = ref(false)
  const currentCI = ref(null)
  const ciCache = ref(new Map())
  
  const searchResults = ref([])
  const searchLoading = ref(false)
  
  const filters = ref({
    type: '',
    status: '',
    search: ''
  })

  // Computed
  const ciOptions = computed(() => 
    cis.value
      .filter(ci => ci.is_active)
      .map(ci => ({
        text: ci.name,
        value: ci.id,
        description: ci.type
      }))
  )
  
  const filteredCIs = computed(() => {
    let result = cis.value
    
    if (filters.value.type) {
      result = result.filter(ci => ci.type === filters.value.type)
    }
    
    if (filters.value.status) {
      result = result.filter(ci => ci.status === filters.value.status)
    }
    
    if (filters.value.search) {
      const searchTerm = filters.value.search.toLowerCase()
      result = result.filter(ci => 
        ci.name.toLowerCase().includes(searchTerm) ||
        ci.type.toLowerCase().includes(searchTerm) ||
        (ci.attributes && Object.values(ci.attributes).some(attr => 
          String(attr).toLowerCase().includes(searchTerm)
        ))
      )
    }
    
    return result
  })

  // Actions
  const setCIs = ({ items, total }) => {
    cis.value = items
    cisTotal.value = total
  }
  
  const setCIsLoading = (loading) => {
    cisLoading.value = loading
  }
  
  const setCurrentCI = (ci) => {
    currentCI.value = ci
    if (ci) {
      ciCache.value.set(ci.id, ci)
    }
  }
  
  const setSearchResults = (results) => {
    searchResults.value = results
  }
  
  const setSearchLoading = (loading) => {
    searchLoading.value = loading
  }
  
  const setFilters = (newFilters) => {
    filters.value = { ...filters.value, ...newFilters }
  }
  
  const updateCIInCache = (ci) => {
    if (ci && ci.id) {
      ciCache.value.set(ci.id, ci)
    }
  }
  
  const removeCIFromCache = (id) => {
    ciCache.value.delete(id)
  }

  // CRUD Actions
  const fetchCIs = async (params = {}) => {
    setCIsLoading(true)
    try {
      const response = await api.get('/api/v1/cis', { params })
      setCIs({
        items: response.data.items,
        total: response.data.total_count
      })
      return response.data
    } finally {
      setCIsLoading(false)
    }
  }
  
  const fetchCI = async (id) => {
    // Check cache first
    if (ciCache.value.has(id)) {
      setCurrentCI(ciCache.value.get(id))
      return ciCache.value.get(id)
    }
    
    try {
      const response = await api.get(`/api/v1/cis/${id}`)
      const ci = response.data
      setCurrentCI(ci)
      return ci
    } catch (error) {
      console.error('Failed to fetch CI:', error)
      throw error
    }
  }
  
  const createCI = async (ciData) => {
    try {
      const response = await api.post('/api/v1/cis', ciData)
      const ci = response.data
      setCurrentCI(ci)
      return ci
    } catch (error) {
      console.error('Failed to create CI:', error)
      throw error
    }
  }
  
  const updateCI = async ({ id, ...ciData }) => {
    try {
      const response = await api.put(`/api/v1/cis/${id}`, ciData)
      const ci = response.data
      setCurrentCI(ci)
      updateCIInCache(ci)
      return ci
    } catch (error) {
      console.error('Failed to update CI:', error)
      throw error
    }
  }
  
  const deleteCI = async (id) => {
    try {
      await api.delete(`/api/v1/cis/${id}`)
      // Remove from cache
      removeCIFromCache(id)
      // Clear current if it's the deleted one
      if (currentCI.value?.id === id) {
        setCurrentCI(null)
      }
      // Remove from list
      cis.value = cis.value.filter(ci => ci.id !== id)
    } catch (error) {
      console.error('Failed to delete CI:', error)
      throw error
    }
  }
  
  // Search Actions
  const searchCIs = async (query, filters = {}) => {
    setSearchLoading(true)
    try {
      const response = await api.get('/api/v1/cis/search', {
        params: { q: query, ...filters }
      })
      setSearchResults(response.data.items)
      return response.data
    } finally {
      setSearchLoading(false)
    }
  }
  
  // Filter Actions
  const applyFilters = (newFilters) => {
    setFilters(newFilters)
  }
  
  const clearFilters = () => {
    setFilters({
      type: '',
      status: '',
      search: ''
    })
  }

  // Utility Getters
  const getCIById = (id) => ciCache.value.get(id)
  const getCIByName = (name) => cis.value.find(ci => ci.name === name)
  const getCIsByType = (type) => cis.value.filter(ci => ci.type === type)
  const getCIsByStatus = (status) => cis.value.filter(ci => ci.status === status)

  return {
    // State
    cis,
    cisTotal,
    cisLoading,
    currentCI,
    ciCache,
    searchResults,
    searchLoading,
    filters,
    
    // Computed
    ciOptions,
    filteredCIs,
    
    // Actions
    setCIs,
    setCIsLoading,
    setCurrentCI,
    setSearchResults,
    setSearchLoading,
    setFilters,
    updateCIInCache,
    removeCIFromCache,
    fetchCIs,
    fetchCI,
    createCI,
    updateCI,
    deleteCI,
    searchCIs,
    applyFilters,
    clearFilters,
    
    // Utility Getters
    getCIById,
    getCIByName,
    getCIsByType,
    getCIsByStatus
  }
})
