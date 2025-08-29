import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook } from '@testing-library/react'
import { useConnectionOptions } from './useConnectionOptions'
import * as canvasStoreModule from '../store/canvasStore'

// Mock the canvas store
const mockUseCanvasStore = vi.fn()

vi.mock('../store/canvasStore', () => ({
  useCanvasStore: vi.fn()
}))

describe('useConnectionOptions', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  const createMockConnection = (name: string, type: 'TYPE_STAGE' | 'TYPE_EVENT_SOURCE' | 'TYPE_CONNECTION_GROUP' = 'TYPE_STAGE') => ({
    name,
    type,
    filters: []
  })

  it('should deduplicate stages with the same name', () => {
    const mockStages = [
      {
        metadata: {
          id: 'stage-1',
          name: 'HTTP party stage'
        }
      },
      {
        metadata: {
          id: 'stage-2', 
          name: 'HTTP party stage' // Duplicate name
        }
      },
      {
        metadata: {
          id: 'stage-3',
          name: 'other stage'
        }
      }
    ]

    mockUseCanvasStore.mockReturnValue({
      stages: mockStages,
      eventSources: [],
      connectionGroups: []
    })
    
    vi.mocked(canvasStoreModule.useCanvasStore).mockImplementation(mockUseCanvasStore)

    const { result } = renderHook(() => useConnectionOptions('current-entity'))
    const options = result.current.getConnectionOptions()

    // Should only have 2 options, not 3 (due to deduplication)
    const stageOptions = options.filter(opt => opt.group === 'Stages')
    expect(stageOptions).toHaveLength(2)

    // Should contain both unique names
    const stageNames = stageOptions.map(opt => opt.value)
    expect(stageNames).toContain('HTTP party stage')
    expect(stageNames).toContain('other stage')

    // Should not have duplicates
    const uniqueNames = [...new Set(stageNames)]
    expect(uniqueNames).toHaveLength(stageNames.length)
  })

  it('should deduplicate event sources with the same name', () => {
    const mockEventSources = [
      {
        metadata: {
          id: 'source-1',
          name: 'My events source'
        }
      },
      {
        metadata: {
          id: 'source-2',
          name: 'My events source' // Duplicate name
        }
      },
      {
        metadata: {
          id: 'source-3', 
          name: 'test'
        }
      }
    ]

    mockUseCanvasStore.mockReturnValue({
      stages: [],
      eventSources: mockEventSources,
      connectionGroups: []
    })
    
    vi.mocked(canvasStoreModule.useCanvasStore).mockImplementation(mockUseCanvasStore)

    const { result } = renderHook(() => useConnectionOptions())
    const options = result.current.getConnectionOptions()

    // Should only have 2 options, not 3 (due to deduplication)
    const eventSourceOptions = options.filter(opt => opt.group === 'Event Sources')
    expect(eventSourceOptions).toHaveLength(2)

    // Should contain both unique names
    const eventSourceNames = eventSourceOptions.map(opt => opt.value)
    expect(eventSourceNames).toContain('My events source')
    expect(eventSourceNames).toContain('test')

    // Should not have duplicates
    const uniqueNames = [...new Set(eventSourceNames)]
    expect(uniqueNames).toHaveLength(eventSourceNames.length)
  })

  it('should deduplicate connection groups with the same name', () => {
    const mockConnectionGroups = [
      {
        metadata: {
          id: 'group-1',
          name: 'Duplicate Group'
        }
      },
      {
        metadata: {
          id: 'group-2',
          name: 'Duplicate Group' // Duplicate name
        }
      },
      {
        metadata: {
          id: 'group-3',
          name: 'Unique Group'
        }
      }
    ]

    mockUseCanvasStore.mockReturnValue({
      stages: [],
      eventSources: [],
      connectionGroups: mockConnectionGroups
    })
    
    vi.mocked(canvasStoreModule.useCanvasStore).mockImplementation(mockUseCanvasStore)

    const { result } = renderHook(() => useConnectionOptions('current-entity'))
    const options = result.current.getConnectionOptions()

    // Should only have 2 options, not 3 (due to deduplication)
    const groupOptions = options.filter(opt => opt.group === 'Connection Groups')
    expect(groupOptions).toHaveLength(2)

    // Should contain both unique names
    const groupNames = groupOptions.map(opt => opt.value)
    expect(groupNames).toContain('Duplicate Group')
    expect(groupNames).toContain('Unique Group')

    // Should not have duplicates
    const uniqueNames = [...new Set(groupNames)]
    expect(uniqueNames).toHaveLength(groupNames.length)
  })

  it('should deduplicate across different entity types', () => {
    const mockStages = [
      {
        metadata: {
          id: 'stage-1',
          name: 'shared-name'
        }
      }
    ]

    const mockEventSources = [
      {
        metadata: {
          id: 'source-1',
          name: 'shared-name' // Same name as stage
        }
      }
    ]

    const mockConnectionGroups = [
      {
        metadata: {
          id: 'group-1', 
          name: 'shared-name' // Same name as stage and event source
        }
      }
    ]

    mockUseCanvasStore.mockReturnValue({
      stages: mockStages,
      eventSources: mockEventSources,
      connectionGroups: mockConnectionGroups
    })
    
    vi.mocked(canvasStoreModule.useCanvasStore).mockImplementation(mockUseCanvasStore)

    const { result } = renderHook(() => useConnectionOptions('current-entity'))
    const options = result.current.getConnectionOptions()

    // Should only have 1 option total due to cross-type deduplication
    expect(options).toHaveLength(1)
    expect(options[0].value).toBe('shared-name')
    expect(options[0].group).toBe('Stages') // First one wins (stages are processed first)
  })

  it('should exclude current entity from stages and connection groups', () => {
    const mockStages = [
      {
        metadata: {
          id: 'current-entity',
          name: 'Current Stage'
        }
      },
      {
        metadata: {
          id: 'other-stage',
          name: 'Other Stage'
        }
      }
    ]

    const mockConnectionGroups = [
      {
        metadata: {
          id: 'current-entity',
          name: 'Current Group'
        }
      },
      {
        metadata: {
          id: 'other-group',
          name: 'Other Group'
        }
      }
    ]

    mockUseCanvasStore.mockReturnValue({
      stages: mockStages,
      eventSources: [],
      connectionGroups: mockConnectionGroups
    })
    
    vi.mocked(canvasStoreModule.useCanvasStore).mockImplementation(mockUseCanvasStore)

    const { result } = renderHook(() => useConnectionOptions('current-entity'))
    const options = result.current.getConnectionOptions()

    // Should only include non-current entities
    expect(options).toHaveLength(2)
    const optionValues = options.map(opt => opt.value)
    expect(optionValues).toContain('Other Stage')
    expect(optionValues).toContain('Other Group')
    expect(optionValues).not.toContain('Current Stage')
    expect(optionValues).not.toContain('Current Group')
  })

  it('should exclude already selected connections', () => {
    const mockStages = [
      {
        metadata: {
          id: 'stage-1',
          name: 'Stage 1'
        }
      },
      {
        metadata: {
          id: 'stage-2',
          name: 'Stage 2'
        }
      },
      {
        metadata: {
          id: 'stage-3',
          name: 'Stage 3'
        }
      }
    ]

    const mockEventSources = [
      {
        metadata: {
          id: 'source-1',
          name: 'Source 1'
        }
      }
    ]

    // Existing connections that should be filtered out
    const existingConnections = [
      createMockConnection('Stage 1'),
      createMockConnection('Source 1', 'TYPE_EVENT_SOURCE')
    ]

    mockUseCanvasStore.mockReturnValue({
      stages: mockStages,
      eventSources: mockEventSources,
      connectionGroups: []
    })
    
    vi.mocked(canvasStoreModule.useCanvasStore).mockImplementation(mockUseCanvasStore)

    const { result } = renderHook(() => useConnectionOptions('current-entity', existingConnections))
    const options = result.current.getConnectionOptions()

    // Should only include options not already selected
    const optionValues = options.map(opt => opt.value)
    expect(optionValues).toContain('Stage 2')
    expect(optionValues).toContain('Stage 3')
    expect(optionValues).not.toContain('Stage 1') // Already selected
    expect(optionValues).not.toContain('Source 1') // Already selected

    expect(options).toHaveLength(2)
  })

  it('should allow editing the current connection (exclude others but include current)', () => {
    const mockStages = [
      {
        metadata: {
          id: 'stage-1',
          name: 'Stage 1'
        }
      },
      {
        metadata: {
          id: 'stage-2',
          name: 'Stage 2'
        }
      }
    ]

    // Existing connections
    const existingConnections = [
      createMockConnection('Stage 1'), // Index 0 - currently being edited
      createMockConnection('Stage 2')  // Index 1 - should be filtered out
    ]

    mockUseCanvasStore.mockReturnValue({
      stages: mockStages,
      eventSources: [],
      connectionGroups: []
    })
    
    vi.mocked(canvasStoreModule.useCanvasStore).mockImplementation(mockUseCanvasStore)

    const { result } = renderHook(() => useConnectionOptions('current-entity', existingConnections))
    
    // When editing connection at index 0
    const optionsForEditingFirst = result.current.getConnectionOptions(0)
    const valuesForEditingFirst = optionsForEditingFirst.map(opt => opt.value)
    
    // Should include Stage 1 (current) but exclude Stage 2 (already selected elsewhere)
    expect(valuesForEditingFirst).toContain('Stage 1')
    expect(valuesForEditingFirst).not.toContain('Stage 2')

    // When editing connection at index 1  
    const optionsForEditingSecond = result.current.getConnectionOptions(1)
    const valuesForEditingSecond = optionsForEditingSecond.map(opt => opt.value)
    
    // Should include Stage 2 (current) but exclude Stage 1 (already selected elsewhere)
    expect(valuesForEditingSecond).toContain('Stage 2')
    expect(valuesForEditingSecond).not.toContain('Stage 1')
  })

  it('should work without existing connections parameter', () => {
    const mockStages = [
      {
        metadata: {
          id: 'stage-1',
          name: 'Stage 1'
        }
      }
    ]

    mockUseCanvasStore.mockReturnValue({
      stages: mockStages,
      eventSources: [],
      connectionGroups: []
    })
    
    vi.mocked(canvasStoreModule.useCanvasStore).mockImplementation(mockUseCanvasStore)

    // Test without existingConnections parameter
    const { result } = renderHook(() => useConnectionOptions('current-entity'))
    const options = result.current.getConnectionOptions()

    expect(options).toHaveLength(1)
    expect(options[0].value).toBe('Stage 1')
  })
})