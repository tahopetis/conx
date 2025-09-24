// Store exports for easy importing
export { useAuthStore } from './auth'
export { useSchemaStore } from './schema'
export { useCIStore } from './ci'
export { useRelationshipStore } from './relationship'

// Store initialization utilities
export const initializeStores = async (app) => {
  // Initialize auth store
  const authStore = useAuthStore(app.$pinia)
  await authStore.initializeAuth()
  
  // You can add other store initializations here if needed
  console.log('All stores initialized')
}

// Store composition utilities
export const useStores = () => {
  const authStore = useAuthStore()
  const schemaStore = useSchemaStore()
  const ciStore = useCIStore()
  const relationshipStore = useRelationshipStore()
  
  return {
    auth: authStore,
    schema: schemaStore,
    ci: ciStore,
    relationship: relationshipStore
  }
}
