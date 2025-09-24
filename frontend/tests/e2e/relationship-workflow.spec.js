import { describe, it, expect, beforeEach, afterEach } from 'vitest'
import { chromium } from 'playwright'

describe('Relationship Management E2E Workflow', () => {
  let browser
  let page

  beforeEach(async () => {
    browser = await chromium.launch()
    page = await browser.newPage()
    
    // Mock API responses
    await page.route('**/api/v1/relationships**', async (route) => {
      if (route.request().method() === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
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
          })
        })
      }
    })

    await page.route('**/api/v1/relationships/1**', async (route) => {
      if (route.request().method() === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
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
          })
        })
      }
    })

    await page.route('**/api/v1/schemas/relationship-types**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          schemas: [
            {
              id: 1,
              name: 'depends_on',
              attributes: [
                { name: 'description', type: 'string', required: false, label: 'Description' },
                { name: 'priority', type: 'number', required: true, label: 'Priority', default: '1' }
              ]
            },
            {
              id: 2,
              name: 'hosted_on',
              attributes: [
                { name: 'port', type: 'number', required: true, label: 'Port' },
                { name: 'is_primary', type: 'boolean', required: false, label: 'Is Primary', default: 'false' }
              ]
            }
          ]
        })
      })
    })

    await page.route('**/api/v1/schemas/relationship-types/1**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: 1,
          name: 'depends_on',
          attributes: [
            { name: 'description', type: 'string', required: false, label: 'Description' },
            { name: 'priority', type: 'number', required: true, label: 'Priority', default: '1' }
          ]
        })
      })
    })

    await page.route('**/api/v1/cis**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [
            { id: 1, name: 'Server 1', type: 'server', environment: 'production', status: 'active' },
            { id: 2, name: 'Database 1', type: 'database', environment: 'production', status: 'active' },
            { id: 3, name: 'Application 1', type: 'application', environment: 'production', status: 'active' }
          ]
        })
      })
    })

    // Navigate to the relationships page
    await page.goto('http://localhost:5173/relationships')
  })

  afterEach(async () => {
    await browser.close()
  })

  it('should display relationships list', async () => {
    // Wait for the page to load
    await page.waitForSelector('h1', { text: 'Relationships' })
    
    // Check if the relationships table is visible
    const table = await page.waitForSelector('.v-data-table')
    expect(table).toBeTruthy()
    
    // Check if the relationship is displayed
    const relationshipRow = await page.waitForSelector('text=Server 1')
    expect(relationshipRow).toBeTruthy()
    
    // Check if schema chip is visible
    const schemaChip = await page.waitForSelector('text=depends_on')
    expect(schemaChip).toBeTruthy()
  })

  it('should filter relationships by schema', async () => {
    // Wait for the page to load
    await page.waitForSelector('h1', { text: 'Relationships' })
    
    // Click on the schema filter dropdown
    await page.click('text=Schema')
    
    // Select the 'depends_on' schema
    await page.click('text=depends_on')
    
    // Wait for the filtered results
    await page.waitForTimeout(1000)
    
    // Check if the filtered relationship is displayed
    const relationshipRow = await page.waitForSelector('text=Server 1')
    expect(relationshipRow).toBeTruthy()
  })

  it('should navigate to relationship details', async () => {
    // Wait for the page to load
    await page.waitForSelector('h1', { text: 'Relationships' })
    
    // Click on the view button for the first relationship
    await page.click('button[color="primary"] >> text=mdi-eye')
    
    // Wait for the details page to load
    await page.waitForSelector('h1', { text: 'Relationship Details' })
    
    // Check if relationship details are displayed
    const sourceCI = await page.waitForSelector('text=Server 1')
    expect(sourceCI).toBeTruthy()
    
    const targetCI = await page.waitForSelector('text=Database 1')
    expect(targetCI).toBeTruthy()
    
    const schemaName = await page.waitForSelector('text=depends_on')
    expect(schemaName).toBeTruthy()
  })

  it('should create a new relationship', async () => {
    // Wait for the page to load
    await page.waitForSelector('h1', { text: 'Relationships' })
    
    // Click on the create button
    await page.click('text=Create Relationship')
    
    // Wait for the create page to load
    await page.waitForSelector('h1', { text: 'Create Relationship' })
    
    // Select source CI
    await page.click('text=Target Configuration Item *')
    await page.click('text=Database 1')
    
    // Select relationship schema
    await page.click('text=Relationship Type Schema *')
    await page.click('text=depends_on')
    
    // Wait for the dynamic form to load
    await page.waitForSelector('text=Description')
    
    // Fill in the form fields
    await page.fill('input[placeholder="Description"]', 'Test relationship description')
    await page.fill('input[placeholder="Priority"]', '2')
    
    // Click the create button
    await page.click('text=Create Relationship')
    
    // Wait for navigation back to relationships list
    await page.waitForSelector('h1', { text: 'Relationships' })
    
    // Check if we're back on the relationships page
    expect(await page.locator('h1').textContent()).toBe('Relationships')
  })

  it('should edit an existing relationship', async () => {
    // Wait for the page to load
    await page.waitForSelector('h1', { text: 'Relationships' })
    
    // Click on the edit button for the first relationship
    await page.click('button[color="warning"] >> text=mdi-pencil')
    
    // Wait for the edit page to load
    await page.waitForSelector('h1', { text: 'Edit Relationship' })
    
    // Wait for the dynamic form to load
    await page.waitForSelector('text=Description')
    
    // Update the description
    await page.fill('input[placeholder="Description"]', 'Updated relationship description')
    
    // Update the priority
    await page.fill('input[placeholder="Priority"]', '3')
    
    // Click the update button
    await page.click('text=Update Relationship')
    
    // Wait for navigation back to relationship details
    await page.waitForSelector('h1', { text: 'Relationship Details' })
    
    // Check if we're on the relationship details page
    expect(await page.locator('h1').textContent()).toBe('Relationship Details')
  })

  it('should delete a relationship', async () => {
    // Wait for the page to load
    await page.waitForSelector('h1', { text: 'Relationships' })
    
    // Click on the delete button for the first relationship
    await page.click('button[color="error"] >> text=mdi-delete')
    
    // Wait for the confirmation dialog
    await page.waitForSelector('text=Confirm Delete')
    
    // Click the delete button in the dialog
    await page.click('text=Delete')
    
    // Wait for the relationships list to reload
    await page.waitForTimeout(1000)
    
    // Check if we're back on the relationships page
    expect(await page.locator('h1').textContent()).toBe('Relationships')
  })

  it('should display relationship visualization', async () => {
    // Wait for the page to load
    await page.waitForSelector('h1', { text: 'Relationships' })
    
    // Click on the view button for the first relationship
    await page.click('button[color="primary"] >> text=mdi-eye')
    
    // Wait for the details page to load
    await page.waitForSelector('h1', { text: 'Relationship Details' })
    
    // Check if the relationship visualization is displayed
    const visualization = await page.waitForSelector('.relationship-visualization')
    expect(visualization).toBeTruthy()
    
    // Check if the source CI node is displayed
    const sourceNode = await page.waitForSelector('text=Source CI')
    expect(sourceNode).toBeTruthy()
    
    // Check if the target CI node is displayed
    const targetNode = await page.waitForSelector('text=Target CI')
    expect(targetNode).toBeTruthy()
    
    // Check if the relationship arrow is displayed
    const arrow = await page.waitForSelector('text=mdi-arrow-right')
    expect(arrow).toBeTruthy()
  })

  it('should handle form validation errors', async () => {
    // Wait for the page to load
    await page.waitForSelector('h1', { text: 'Relationships' })
    
    // Click on the create button
    await page.click('text=Create Relationship')
    
    // Wait for the create page to load
    await page.waitForSelector('h1', { text: 'Create Relationship' })
    
    // Try to submit the form without filling required fields
    await page.click('text=Create Relationship')
    
    // Check if validation errors are displayed
    const errorMessages = await page.locator('.v-messages__message')
    expect(await errorMessages.count()).toBeGreaterThan(0)
  })

  it('should navigate back from create form', async () => {
    // Wait for the page to load
    await page.waitForSelector('h1', { text: 'Relationships' })
    
    // Click on the create button
    await page.click('text=Create Relationship')
    
    // Wait for the create page to load
    await page.waitForSelector('h1', { text: 'Create Relationship' })
    
    // Click the cancel button
    await page.click('text=Cancel')
    
    // Wait for navigation back to relationships list
    await page.waitForSelector('h1', { text: 'Relationships' })
    
    // Check if we're back on the relationships page
    expect(await page.locator('h1').textContent()).toBe('Relationships')
  })

  it('should display loading states', async () => {
    // Mock a slow API response
    await page.route('**/api/v1/relationships**', async (route) => {
      await new Promise(resolve => setTimeout(resolve, 2000))
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [],
          total: 0,
          page: 1,
          limit: 20
        })
      })
    })

    // Navigate to the relationships page
    await page.goto('http://localhost:5173/relationships')
    
    // Check if loading skeleton is displayed
    const skeletonLoader = await page.waitForSelector('.v-skeleton-loader')
    expect(skeletonLoader).toBeTruthy()
  })

  it('should handle API errors gracefully', async () => {
    // Mock an API error
    await page.route('**/api/v1/relationships**', async (route) => {
      await route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ message: 'Internal Server Error' })
      })
    })

    // Navigate to the relationships page
    await page.goto('http://localhost:5173/relationships')
    
    // Wait for the page to load
    await page.waitForTimeout(1000)
    
    // Check if error message is displayed
    const errorAlert = await page.locator('.v-alert')
    expect(await errorAlert.count()).toBeGreaterThan(0)
  })

  it('should be responsive on mobile devices', async () => {
    // Set viewport to mobile size
    await page.setViewportSize({ width: 375, height: 667 })
    
    // Navigate to the relationships page
    await page.goto('http://localhost:5173/relationships')
    
    // Wait for the page to load
    await page.waitForSelector('h1', { text: 'Relationships' })
    
    // Check if the mobile menu button is visible
    const mobileMenu = await page.locator('.v-app-bar-nav-icon')
    expect(await mobileMenu.isVisible()).toBe(true)
    
    // Check if the content is properly laid out
    const table = await page.locator('.v-data-table')
    expect(await table.isVisible()).toBe(true)
  })

  it('should maintain state during navigation', async () => {
    // Wait for the page to load
    await page.waitForSelector('h1', { text: 'Relationships' })
    
    // Apply a filter
    await page.click('text=Schema')
    await page.click('text=depends_on')
    
    // Wait for the filter to apply
    await page.waitForTimeout(1000)
    
    // Navigate to relationship details
    await page.click('button[color="primary"] >> text=mdi-eye')
    
    // Wait for the details page to load
    await page.waitForSelector('h1', { text: 'Relationship Details' })
    
    // Navigate back to relationships list
    await page.click('text=Back to Relationships')
    
    // Wait for the relationships page to load
    await page.waitForSelector('h1', { text: 'Relationships' })
    
    // Check if the filter is still applied
    const schemaFilter = await page.locator('text=depends_on')
    expect(await schemaFilter.isVisible()).toBe(true)
  })
})
