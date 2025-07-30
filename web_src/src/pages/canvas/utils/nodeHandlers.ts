import { useCanvasStore } from '../store/canvasStore';
import { NodeType, createEmptyNode, CreateNodeParams } from './nodeFactories';
import { EventSourceWithEvents } from '../store/types';

/**
 * Hook that provides modular node handling functionality
 */
export const useNodeHandlers = (canvasId: string) => {
  const { addStage, addEventSource, addConnectionGroup } = useCanvasStore();

  /**
   * Handles adding a new node of the specified type
   */
  const handleAddNode = (nodeType: NodeType, customParams?: Partial<CreateNodeParams>) => {
    const params: CreateNodeParams = {
      canvasId,
      ...customParams
    };

    try {
      switch (nodeType) {
        case 'stage': {
          const stage = createEmptyNode('stage', params);
          addStage(stage, true); // true = draft mode
          break;
        }
        
        case 'event_source': {
          const eventSource = createEmptyNode('event_source', params);
          // addEventSource expects EventSourceWithEvents, so we need to add the events property
          const eventSourceWithEvents: EventSourceWithEvents = {
            ...eventSource,
            events: []
          };
          addEventSource(eventSourceWithEvents);
          break;
        }
        
        case 'connection_group': {
          const connectionGroup = createEmptyNode('connection_group', params);
          addConnectionGroup(connectionGroup);
          break;
        }
        
        default:
          console.error(`Unknown node type: ${nodeType}`);
          throw new Error(`Cannot add node of unknown type: ${nodeType}`);
      }
    } catch (error) {
      console.error(`Failed to add ${nodeType}:`, error);
      throw error;
    }
  };

  /**
   * Handles adding multiple nodes at once
   */
  const handleAddNodes = (nodeConfigs: Array<{ type: NodeType; params?: Partial<CreateNodeParams> }>) => {
    nodeConfigs.forEach(({ type, params }) => {
      handleAddNode(type, params);
    });
  };

  /**
   * Handles adding a node with a custom name
   */
  const handleAddNamedNode = (nodeType: NodeType, name: string) => {
    handleAddNode(nodeType, { name });
  };

  /**
   * Handles adding a stage with a specific executor type
   */
  const handleAddStageWithExecutor = (name: string, executorType: string) => {
    handleAddNode('stage', { name, executorType });
  };

  return {
    handleAddNode,
    handleAddNodes,
    handleAddNamedNode,
    handleAddStageWithExecutor
  };
};