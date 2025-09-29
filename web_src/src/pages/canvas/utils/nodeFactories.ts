import { SuperplaneStage, SuperplaneEventSource, SuperplaneConnectionGroup } from '@/api-client/types.gen';

/**
 * Factory functions to create empty/default nodes for different types
 */

export type NodeType = 'stage' | 'event_source' | 'connection_group';


export interface CreateNodeParams {
  canvasId: string;
  name?: string;
  spec?: any;
}

/**
 * Creates an empty stage with default configuration
 */
export const createEmptyStage = ({ canvasId, name = 'New Stage', spec }: CreateNodeParams): SuperplaneStage => {
  const executorType = spec?.executor?.type;

  const getExecutorTemplate = (type?: string) => {
    switch (type) {
      case 'semaphore':
        return {
          type: 'semaphore',
          integration: {
            name: '',
          },
          resource: {
            type: 'project',
            name: '',
          },
          spec: {
            ref: 'refs/heads/main',
            pipelineFile: '.semaphore/semaphore.yml',
          }
        };
      case 'github':
        return {
          type: 'github',
          integration: {
            name: '',
          },
          resource: {
            type: 'repository',
            name: '',
          },
          spec: {
            ref: 'main',
            workflow: '.github/workflows/main.yml',
          }
        };
      case 'http':
        return {
          type: 'http',
          spec: {
            url: '',
          },
        };
      case 'noop':
        return {
          type: 'noop',
          spec: {},
        };
      default:
        return { type: '', spec: {} };
    }
  };

  return {
    metadata: {
      canvasId,
      name,
      id: Date.now().toString(), // Temporary ID
    },
    spec: {
      conditions: [],
      inputs: [],
      outputs: [],
      executor: getExecutorTemplate(executorType),
      connections: spec?.connections || [],
      inputMappings: [],
      secrets: [],
      dryRun: true
    },
  };
};

/**
 * Creates an empty event source with default configuration
 */
export const createEmptyEventSource = ({ canvasId, name = 'New Event Source', spec }: CreateNodeParams): SuperplaneEventSource => {
  return {
    metadata: {
      canvasId,
      name,
      id: Date.now().toString(), // Temporary ID
    },
    spec: {
      type: spec?.type || 'webhook'
    }
  };
};

/**
 * Creates an empty connection group with default configuration
 */
export const createEmptyConnectionGroup = ({ canvasId, name = 'New Connection Group', spec }: CreateNodeParams): SuperplaneConnectionGroup => {
  const connections = spec?.connections || [];
  return {
    metadata: {
      canvasId,
      name,
      id: Date.now().toString(), // Temporary ID
    },
    spec: {
      connections: connections,
      groupBy: {
        // Default groupBy configuration
      }
    }
  };
};

/**
 * Factory function that creates the appropriate empty node based on type
 * Using function overloads to ensure type safety
 */
export function createEmptyNode(nodeType: 'stage', params: CreateNodeParams & { executorType?: string }): SuperplaneStage;
export function createEmptyNode(nodeType: 'event_source', params: CreateNodeParams): SuperplaneEventSource;
export function createEmptyNode(nodeType: 'connection_group', params: CreateNodeParams): SuperplaneConnectionGroup;
export function createEmptyNode(nodeType: NodeType, params: CreateNodeParams): SuperplaneStage | SuperplaneEventSource | SuperplaneConnectionGroup {
  switch (nodeType) {
    case 'stage':
      return createEmptyStage(params);
    case 'event_source':
      return createEmptyEventSource(params);
    case 'connection_group':
      return createEmptyConnectionGroup(params);
    default:
      throw new Error(`Unknown node type: ${nodeType}`);
  }
}
