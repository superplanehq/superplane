import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ConnectionSelector } from './ConnectionSelector'
import { SuperplaneConnection } from '@/api-client/types.gen'
import * as useConnectionOptionsModule from '../../hooks/useConnectionOptions'

// Mock the useConnectionOptions hook
const mockGetConnectionOptions = vi.fn()

vi.mock('../../hooks/useConnectionOptions', () => ({
  useConnectionOptions: vi.fn()
}))

const mockConnection: SuperplaneConnection = {
  name: 'test-connection',
  type: 'TYPE_STAGE',
  filters: []
}

const mockProps = {
  connection: mockConnection,
  index: 0,
  onConnectionUpdate: vi.fn(),
  onFilterAdd: vi.fn(),
  onFilterUpdate: vi.fn(),
  onFilterRemove: vi.fn(),
  onFilterOperatorToggle: vi.fn(),
  currentEntityId: 'current-entity-id'
}

describe('ConnectionSelector', () => {
  beforeEach(() => {
    vi.clearAllMocks()

    // Setup the mock implementation
    vi.mocked(useConnectionOptionsModule.useConnectionOptions).mockReturnValue({
      getConnectionOptions: mockGetConnectionOptions
    })
  })

  it('should use deduplicated options from useConnectionOptions hook', async () => {
    const user = userEvent.setup()

    // Setup mock data that's already deduplicated (as our hook should provide)
    const mockDeduplicatedOptions = [
      {
        value: 'HTTP party stage',
        label: 'HTTP party stage',
        group: 'Stages',
        type: 'TYPE_STAGE' as const
      },
      {
        value: 'other',
        label: 'other',
        group: 'Stages',
        type: 'TYPE_STAGE' as const
      },
      {
        value: 'My events source',
        label: 'My events source',
        group: 'Event Sources',
        type: 'TYPE_EVENT_SOURCE' as const
      },
      {
        value: 'test',
        label: 'test',
        group: 'Event Sources',
        type: 'TYPE_EVENT_SOURCE' as const
      }
    ]

    mockGetConnectionOptions.mockReturnValue(mockDeduplicatedOptions)

    render(<ConnectionSelector {...mockProps} />)

    // Focus the combobox input to open the dropdown
    const combobox = screen.getByRole('combobox')
    await user.click(combobox)

    // Now get all options within the dropdown
    const allOptions = screen.getAllByRole('option')

    // Should have 4 unique options
    expect(allOptions).toHaveLength(4)

    // Extract the text content of all options
    const optionTexts = allOptions.map((option: HTMLElement) => option.textContent?.trim()).filter(Boolean) as string[]

    // Check that there are no duplicate option texts
    const uniqueOptionTexts = [...new Set(optionTexts)]
    expect(uniqueOptionTexts).toHaveLength(optionTexts.length)

    // Verify specific expected options are present
    expect(optionTexts).toContain('HTTP party stage')
    expect(optionTexts).toContain('other')
    expect(optionTexts).toContain('My events source')
    expect(optionTexts).toContain('test')

    // Count occurrences of "HTTP party stage" - should only appear once
    const httpPartyStageCount = optionTexts.filter((text: string) => text === 'HTTP party stage').length
    expect(httpPartyStageCount).toBe(1)
  })

  it('should render correct option groups', async () => {
    const user = userEvent.setup()

    const mockOptions = [
      {
        value: 'stage1',
        label: 'Stage 1',
        group: 'Stages',
        type: 'TYPE_STAGE' as const
      },
      {
        value: 'source1',
        label: 'Source 1',
        group: 'Event Sources',
        type: 'TYPE_EVENT_SOURCE' as const
      }
    ]

    mockGetConnectionOptions.mockReturnValue(mockOptions)

    render(<ConnectionSelector {...mockProps} />)

    // Focus the combobox input to open the dropdown
    const combobox = screen.getByRole('combobox')
    await user.click(combobox)

    // Check that group labels are rendered (these appear as text content, not role="group")
    expect(screen.getByText('Stages')).toBeInTheDocument()
    expect(screen.getByText('Event Sources')).toBeInTheDocument()

    // Check that options are rendered
    expect(screen.getByRole('option', { name: 'Stage 1' })).toBeInTheDocument()
    expect(screen.getByRole('option', { name: 'Source 1' })).toBeInTheDocument()
  })

  it('should render empty state when no connections are available', async () => {
    const user = userEvent.setup()

    mockGetConnectionOptions.mockReturnValue([])

    const connectionWithType = { ...mockConnection, type: 'TYPE_STAGE' as const }

    render(<ConnectionSelector {...mockProps} connection={connectionWithType} />)

    // Focus the combobox input to open the dropdown
    const combobox = screen.getByRole('combobox')
    await user.click(combobox)

    expect(screen.getByText('No connections available')).toBeInTheDocument()
  })

  it('should not render filters section when showFilters is false', () => {
    mockGetConnectionOptions.mockReturnValue([])

    render(<ConnectionSelector {...mockProps} showFilters={false} />)

    expect(screen.queryByText('Filters')).not.toBeInTheDocument()
  })

  it('should render filters section when showFilters is true', () => {
    mockGetConnectionOptions.mockReturnValue([])

    render(<ConnectionSelector {...mockProps} showFilters={true} />)

    expect(screen.getByText('Filters')).toBeInTheDocument()
  })

  it('should pass existing connections to useConnectionOptions hook', () => {
    const existingConnections = [
      { name: 'Connection 1', type: 'TYPE_STAGE' as const, filters: [] },
      { name: 'Connection 2', type: 'TYPE_EVENT_SOURCE' as const, filters: [] }
    ]

    mockGetConnectionOptions.mockReturnValue([])

    render(
      <ConnectionSelector
        {...mockProps}
        existingConnections={existingConnections}
      />
    )

    expect(useConnectionOptionsModule.useConnectionOptions).toHaveBeenCalledWith(
      mockProps.currentEntityId,
      existingConnections
    )
  })

  it('should pass current connection index to getConnectionOptions', () => {
    const mockOptions = [
      {
        value: 'available-option',
        label: 'Available Option',
        group: 'Stages',
        type: 'TYPE_STAGE' as const
      }
    ]

    mockGetConnectionOptions.mockReturnValue(mockOptions)

    render(<ConnectionSelector {...mockProps} index={2} />)

    expect(mockGetConnectionOptions).toHaveBeenCalledWith(2)
  })

  it('should not render existing connections as options', async () => {
    const user = userEvent.setup()

    const existingConnections = [
      { name: 'Already Selected Stage', type: 'TYPE_STAGE' as const, filters: [] },
      { name: 'Already Selected Source', type: 'TYPE_EVENT_SOURCE' as const, filters: [] }
    ]

    const mockAvailableOptions = [
      {
        value: 'Available Stage 1',
        label: 'Available Stage 1',
        group: 'Stages',
        type: 'TYPE_STAGE' as const
      },
      {
        value: 'Available Stage 2',
        label: 'Available Stage 2',
        group: 'Stages',
        type: 'TYPE_STAGE' as const
      },
      {
        value: 'Available Source',
        label: 'Available Source',
        group: 'Event Sources',
        type: 'TYPE_EVENT_SOURCE' as const
      }
    ]

    mockGetConnectionOptions.mockReturnValue(mockAvailableOptions)

    render(
      <ConnectionSelector
        {...mockProps}
        existingConnections={existingConnections}
      />
    )

    // Focus the combobox input to open the dropdown
    const combobox = screen.getByRole('combobox')
    await user.click(combobox)

    const allOptions = screen.getAllByRole('option')
    const optionTexts = allOptions
      .map((option: HTMLElement) => option.textContent?.trim())
      .filter(Boolean) as string[]

    expect(optionTexts).toContain('Available Stage 1')
    expect(optionTexts).toContain('Available Stage 2')
    expect(optionTexts).toContain('Available Source')

    expect(optionTexts).not.toContain('Already Selected Stage')
    expect(optionTexts).not.toContain('Already Selected Source')

    expect(optionTexts).toHaveLength(3)
  })

  it('should allow current connection to be selected when editing', async () => {
    const user = userEvent.setup()

    const existingConnections = [
      { name: 'Current Connection', type: 'TYPE_STAGE' as const, filters: [] }, // Index 0 - being edited
      { name: 'Other Connection', type: 'TYPE_STAGE' as const, filters: [] }    // Index 1 - should be excluded
    ]

    const mockOptionsForCurrentEdit = [
      {
        value: 'Current Connection',
        label: 'Current Connection',
        group: 'Stages',
        type: 'TYPE_STAGE' as const
      },
      {
        value: 'Available Connection',
        label: 'Available Connection',
        group: 'Stages',
        type: 'TYPE_STAGE' as const
      }
    ]

    mockGetConnectionOptions.mockReturnValue(mockOptionsForCurrentEdit)

    render(
      <ConnectionSelector
        {...mockProps}
        connection={{ name: 'Current Connection', type: 'TYPE_STAGE', filters: [] }}
        index={0}
        existingConnections={existingConnections}
      />
    )

    // Focus the combobox input to open the dropdown
    const combobox = screen.getByRole('combobox')
    await user.click(combobox)

    // Check that the specific options are present
    expect(screen.getByRole('option', { name: /Current Connection/ })).toBeInTheDocument()
    expect(screen.getByRole('option', { name: /Available Connection/ })).toBeInTheDocument()

    // Check that Other Connection is not present
    expect(screen.queryByRole('option', { name: /Other Connection/ })).not.toBeInTheDocument()
    expect(mockGetConnectionOptions).toHaveBeenCalledWith(0)
  })
})