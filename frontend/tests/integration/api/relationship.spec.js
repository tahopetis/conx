import { describe, it, expect, beforeEach, vi } from 'vitest'
import axios from 'axios'
import MockAdapter from 'axios-mock-adapter'
import { relationshipStore } from '@/stores/relationship'

// Mock axios
const mockAxios = new MockAdapter(axios)

describe('Relationship API Integration', () => {
  let store

  beforeEach(() => {
    vi.clearAllMocks()
    mockAxios.reset()
    
    // Create a fresh store instance for each test
    store = relationshipStore()
  })

  describe('fetchRelationships', () => {
    it('fetches relationships with default parameters', async () => {
      const mockResponse = {
        data: [
          {
            id: 1,
            source_id: 1,
            target_id: 2,
            schema_id: 1,
            schema_name: 'depends_on',
            source_ci_name: 'Server 1',
            target_ci_name: 'Database 1',
            direction: 'forward',
            created_at: '2023-01-01T00:00:00Z',
            updated_at: '2023-01-01T00:00:00Z'
          }
        ],
        total: 1,
        page: 1,
        limit: 20
      }

      mockAxios.onGet('/api/v1/relationships').reply(200, mockResponse)

      const result = await store.fetchRelationships()

      expect(result).toEqual(mockResponse)
      expect(mockAxios.history.get[0].params).toEqual({
        page: 1,
        limit: 20,
        sort: 'created_at',
        order: 'desc'
      })
    })

    it('fetches relationships with custom parameters', async () => {
      const mockResponse = {
        data: [],
        total: 0,
        page: 1,
        limit: 10
      }

      const customParams = {
        page: 2,
        limit: 10,
        sort: 'updated_at',
        order: 'asc',
        search: 'test',
        schema_id: 1,
        source_ci_id: 1,
        target_ci_id: 2,
        direction: 'forward'
      }

      mockAxios.onGet('/api/v1/relationships').reply(200, mockResponse)

      const result = await store.fetchRelationships(customParams)

      expect(result).toEqual(mockResponse)
      expect(mockAxios.history.get[0].params).toEqual(customParams)
    })

    it('handles API errors gracefully', async () => {
      mockAxios.onGet('/api/v1/relationships').reply(500, {
        message: 'Internal Server Error'
      })

      await expect(store.fetchRelationships()).rejects.toThrow()
    })
  })

  describe('fetchRelationshipDetail', () => {
    it('fetches single relationship by ID', async () => {
      const mockResponse = {
        id: 1,
        source_id: 1,
        target_id: 2,
        schema_id: 1,
        schema_name: 'depends_on',
        source_ci_name: 'Server 1',
        target_ci_name: 'Database 1',
        direction: 'forward',
        description: 'Test relationship',
        priority: 1,
        created_at: '2023-01-01T00:00:00Z',
        updated_at: '2023-01-01T00:00:00Z'
      }

      mockAxios.onGet('/api/v1/relationships/1').reply(200, mockResponse)

      const result = await store.fetchRelationshipDetail(1)

      expect(result).toEqual(mockResponse)
      expect(mockAxios.history.get[0].url).toBe('/api/v1/relationships/1')
    })

    it('handles 404 error when relationship not found', async () => {
      mockAxios.onGet('/api/v1/relationships/999').reply(404, {
        message: 'Relationship not found'
      })

      await expect(store.fetchRelationshipDetail(999)).rejects.toThrow()
    })
  })

  describe('createRelationship', () => {
    it('creates new relationship successfully', async () => {
      const relationshipData = {
        source_id: 1,
        target_id: 2,
        schema_id: 1,
        schema_name: 'depends_on',
        description: 'Test relationship',
        priority: 1
      }

      const mockResponse = {
        id: 1,
        ...relationshipData,
        created_at: '2023-01-01T00:00:00Z',
        updated_at: '2023-01-01T00:00:00Z'
      }

      mockAxios.onPost('/api/v1/relationships').reply(201, mockResponse)

      const result = await store.createRelationship(relationshipData)

      expect(result).toEqual(mockResponse)
      expect(mockAxios.history.post[0].data).toEqual(JSON.stringify(relationshipData))
    })

    it('handles validation errors', async () => {
      const relationshipData = {
        source_id: 1,
        target_id: 2,
        schema_id: 1,
        schema_name: 'depends_on'
      }

      mockAxios.onPost('/api/v1/relationships').reply(400, {
        message: 'Validation failed',
        errors: {
          description: 'Description is required'
        }
      })

      await expect(store.createRelationship(relationshipData)).rejects.toThrow()
    })
  })

  describe('updateRelationship', () => {
    it('updates existing relationship successfully', async () => {
      const relationshipData = {
        source_id: 1,
        target_id: 2,
        schema_id: 1,
        schema_name: 'depends_on',
        description: 'Updated relationship',
        priority: 2
      }

      const mockResponse = {
        id: 1,
        ...relationshipData,
        created_at: '2023-01-01T00:00:00Z',
        updated_at: '2023-01-02T00:00:00Z'
      }

      mockAxios.onPut('/api/v1/relationships/1').reply(200, mockResponse)

      const result = await store.updateRelationship({
        id: 1,
        data: relationshipData
      })

      expect(result).toEqual(mockResponse)
      expect(mockAxios.history.put[0].url).toBe('/api/v1/relationships/1')
      expect(mockAxios.history.put[0].data).toEqual(JSON.stringify(relationshipData))
    })

    it('handles 404 error when updating non-existent relationship', async () => {
      mockAxios.onPut('/api/v1/relationships/999').reply(404, {
        message: 'Relationship not found'
      })

      await expect(store.updateRelationship({
        id: 999,
        data: { description: 'Updated' }
      })).rejects.toThrow()
    })
  })

  describe('deleteRelationship', () => {
    it('deletes relationship successfully', async () => {
      mockAxios.onDelete('/api/v1/relationships/1').reply(204)

      const result = await store.deleteRelationship(1)

      expect(result).toBeUndefined()
      expect(mockAxios.history.delete[0].url).toBe('/api/v1/relationships/1')
    })

    it('handles 404 error when deleting non-existent relationship', async () => {
      mockAxios.onDelete('/api/v1/relationships/999').reply(404, {
        message: 'Relationship not found'
      })

      await expect(store.deleteRelationship(999)).rejects.toThrow()
    })
  })

  describe('fetchRelatedRelationships', () => {
    it('fetches related relationships successfully', async () => {
      const mockResponse = {
        data: [
          {
            id: 2,
            source_id: 1,
            target_id: 3,
            schema_id: 2,
            schema_name: 'hosted_on',
            source_ci_name: 'Server 1',
            target_ci_name: 'Application 1',
            direction: 'forward',
            related_ci_name: 'Application 1',
            direction: 'source'
          }
        ],
        total: 1
      }

      const params = {
        source_id: 1,
        target_id: 2,
        exclude_id: 1
      }

      mockAxios.onGet('/api/v1/relationships/related').reply(200, mockResponse)

      const result = await store.fetchRelatedRelationships(params)

      expect(result).toEqual(mockResponse)
      expect(mockAxios.history.get[0].params).toEqual(params)
    })
  })

  describe('fetchCIRelationships', () => {
    it('fetches CI relationships for graph visualization', async () => {
      const mockResponse = {
        data: {
          nodes: [
            {
              id: 1,
              name: 'Server 1',
              type: 'server',
              environment: 'production',
              status: 'active',
              owner: 'admin'
            },
            {
              id: 2,
              name: 'Database 1',
              type: 'database',
              environment: 'production',
              status: 'active',
              owner: 'admin'
            }
          ],
          edges: [
            {
              source_id: 1,
              target_id: 2,
              relationship_type: 'depends_on',
              schema_name: 'depends_on'
            }
          ]
        }
      }

      const params = {
        search: 'server',
        type: 'server',
        environment: 'production',
        relationship_type: 'depends_on',
        schema_id: 1
      }

      mockAxios.onGet('/api/v1/relationships/graph').reply(200, mockResponse)

      const result = await store.fetchCIRelationships(params)

      expect(result).toEqual(mockResponse)
      expect(mockAxios.history.get[0].params).toEqual(params)
    })
  })

  describe('fetchAuditHistory', () => {
    it('fetches audit history for a relationship', async () => {
      const mockResponse = {
        data: [
          {
            id: 1,
            relationship_id: 1,
            event_type: 'created',
            description: 'Relationship created',
            user: 'admin',
            created_at: '2023-01-01T00:00:00Z'
          },
          {
            id: 2,
            relationship_id: 1,
            event_type: 'updated',
            description: 'Relationship updated',
            user: 'admin',
            created_at: '2023-01-02T00:00:00Z'
          }
        ]
      }

      mockAxios.onGet('/api/v1/relationships/1/audit').reply(200, mockResponse)

      const result = await store.fetchAuditHistory(1)

      expect(result).toEqual(mockResponse)
      expect(mockAxios.history.get[0].url).toBe('/api/v1/relationships/1/audit')
    })
  })

  describe('error handling', () => {
    it('handles network errors', async () => {
      mockAxios.onGet('/api/v1/relationships').networkError()

      await expect(store.fetchRelationships()).rejects.toThrow('Network Error')
    })

    it('handles timeout errors', async () => {
      mockAxios.onGet('/api/v1/relationships').timeout()

      await expect(store.fetchRelationships()).rejects.toThrow()
    })

    it('handles unauthorized errors', async () => {
      mockAxios.onGet('/api/v1/relationships').reply(401, {
        message: 'Unauthorized'
      })

      await expect(store.fetchRelationships()).rejects.toThrow()
    })

    it('handles forbidden errors', async () => {
      mockAxios.onGet('/api/v1/relationships').reply(403, {
        message: 'Forbidden'
      })

      await expect(store.fetchRelationships()).rejects.toThrow()
    })
  })

  describe('request configuration', () => {
    it('includes authorization headers', async () => {
      // Mock localStorage for auth token
      const localStorageMock = {
        getItem: vi.fn().mockReturnValue('test-token')
      }
      global.localStorage = localStorageMock

      mockAxios.onGet('/api/v1/relationships').reply(200, {
        data: [],
        total: 0,
        page: 1,
        limit: 20
      })

      await store.fetchRelationships()

      expect(mockAxios.history.get[0].headers.Authorization).toBe('Bearer test-token')
    })

    it('includes content-type headers for POST/PUT requests', async () => {
      mockAxios.onPost('/api/v1/relationships').reply(201, {})

      await store.createRelationship({
        source_id: 1,
        target_id: 2,
        schema_id: 1,
        schema_name: 'depends_on'
      })

      expect(mockAxios.history.post[0].headers['Content-Type']).toBe('application/json')
    })
  })
})
