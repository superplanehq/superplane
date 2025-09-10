import { describe, it, expect, beforeEach } from 'vitest';
import { useCanvasStore } from './canvasStore';
import { StageWithEventQueue, ConnectionGroupWithEvents } from './types';

describe('canvasStore', () => {
  beforeEach(() => {
    useCanvasStore.setState({
      stages: [],
      connectionGroups: [],
      eventSources: [],
      canvas: {},
      canvasId: '',
      nodePositions: {},
      eventSourceKeys: {},
      selectedStageId: null,
      selectedEventSourceId: null,
      focusedNodeId: null,
      editingStageId: null,
      editingEventSourceId: null,
      editingConnectionGroupId: null,
      webSocketConnectionStatus: 0,
      nodes: [],
      edges: [],
      handleDragging: undefined,
      lockedNodes: true,
    });
  });

  describe('updateConnectionSourceNames', () => {
    it('should update stage connections that reference the renamed entity', () => {
      const mockStage: StageWithEventQueue = {
        metadata: { id: 'stage1', name: 'Stage 1' },
        spec: {
          connections: [
            { name: 'old-name', type: 'TYPE_EVENT_SOURCE' },
            { name: 'other-name', type: 'TYPE_STAGE' }
          ]
        },
        queue: [],
        events: []
      };

      useCanvasStore.setState({ stages: [mockStage] });

      useCanvasStore.getState().updateConnectionSourceNames('old-name', 'new-name');

      const updatedStages = useCanvasStore.getState().stages;
      expect(updatedStages[0].spec?.connections?.[0].name).toBe('new-name');
      expect(updatedStages[0].spec?.connections?.[1].name).toBe('other-name');
    });

    it('should update connection group connections that reference the renamed entity', () => {
      const mockConnectionGroup: ConnectionGroupWithEvents = {
        metadata: { id: 'cg1', name: 'Connection Group 1' },
        spec: {
          connections: [
            { name: 'old-name', type: 'TYPE_EVENT_SOURCE' },
            { name: 'other-name', type: 'TYPE_STAGE' }
          ]
        },
        events: []
      };

      useCanvasStore.setState({ connectionGroups: [mockConnectionGroup] });

      useCanvasStore.getState().updateConnectionSourceNames('old-name', 'new-name');

      const updatedConnectionGroups = useCanvasStore.getState().connectionGroups;
      expect(updatedConnectionGroups[0].spec?.connections?.[0].name).toBe('new-name');
      expect(updatedConnectionGroups[0].spec?.connections?.[1].name).toBe('other-name');
    });

    it('should update multiple entities with connections referencing the renamed entity', () => {
      const mockStage: StageWithEventQueue = {
        metadata: { id: 'stage1', name: 'Stage 1' },
        spec: {
          connections: [
            { name: 'old-name', type: 'TYPE_EVENT_SOURCE' }
          ]
        },
        queue: [],
        events: []
      };

      const mockConnectionGroup: ConnectionGroupWithEvents = {
        metadata: { id: 'cg1', name: 'Connection Group 1' },
        spec: {
          connections: [
            { name: 'old-name', type: 'TYPE_STAGE' }
          ]
        },
        events: []
      };

      useCanvasStore.setState({ 
        stages: [mockStage],
        connectionGroups: [mockConnectionGroup]
      });

      useCanvasStore.getState().updateConnectionSourceNames('old-name', 'new-name');

      const updatedStages = useCanvasStore.getState().stages;
      const updatedConnectionGroups = useCanvasStore.getState().connectionGroups;
      
      expect(updatedStages[0].spec?.connections?.[0].name).toBe('new-name');
      expect(updatedConnectionGroups[0].spec?.connections?.[0].name).toBe('new-name');
    });

    it('should not modify connections that do not match the old name', () => {
      const mockStage: StageWithEventQueue = {
        metadata: { id: 'stage1', name: 'Stage 1' },
        spec: {
          connections: [
            { name: 'different-name', type: 'TYPE_EVENT_SOURCE' },
            { name: 'another-name', type: 'TYPE_STAGE' }
          ]
        },
        queue: [],
        events: []
      };

      useCanvasStore.setState({ stages: [mockStage] });

      useCanvasStore.getState().updateConnectionSourceNames('old-name', 'new-name');

      const updatedStages = useCanvasStore.getState().stages;
      expect(updatedStages[0].spec?.connections?.[0].name).toBe('different-name');
      expect(updatedStages[0].spec?.connections?.[1].name).toBe('another-name');
    });

    it('should handle entities with no connections', () => {
      const mockStage: StageWithEventQueue = {
        metadata: { id: 'stage1', name: 'Stage 1' },
        spec: {},
        queue: [],
        events: []
      };

      const mockConnectionGroup: ConnectionGroupWithEvents = {
        metadata: { id: 'cg1', name: 'Connection Group 1' },
        spec: {
          connections: undefined
        },
        events: []
      };

      useCanvasStore.setState({ 
        stages: [mockStage],
        connectionGroups: [mockConnectionGroup]
      });

      expect(() => {
        useCanvasStore.getState().updateConnectionSourceNames('old-name', 'new-name');
      }).not.toThrow();
    });

    it('should handle empty state', () => {
      expect(() => {
        useCanvasStore.getState().updateConnectionSourceNames('old-name', 'new-name');
      }).not.toThrow();
    });

    it('should preserve other connection properties when updating name', () => {
      const mockStage: StageWithEventQueue = {
        metadata: { id: 'stage1', name: 'Stage 1' },
        spec: {
          connections: [
            { 
              name: 'old-name', 
              type: 'TYPE_EVENT_SOURCE',
              filters: [{ type: 'FILTER_TYPE_DATA', data: { expression: 'test' } }]
            }
          ]
        },
        queue: [],
        events: []
      };

      useCanvasStore.setState({ stages: [mockStage] });

      useCanvasStore.getState().updateConnectionSourceNames('old-name', 'new-name');

      const updatedStages = useCanvasStore.getState().stages;
      const connection = updatedStages[0].spec?.connections?.[0];
      expect(connection?.name).toBe('new-name');
      expect(connection?.type).toBe('TYPE_EVENT_SOURCE');
      expect(connection?.filters).toEqual([{ type: 'FILTER_TYPE_DATA', data: { expression: 'test' } }]);
    });
  });
});