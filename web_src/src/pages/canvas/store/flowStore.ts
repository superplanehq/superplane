// This module contains callbacks for handling events from the ReactFlow component
// and manages the state of the displayed flow

import { create } from 'zustand';
import { applyNodeChanges, applyEdgeChanges, Connection, Viewport } from '@xyflow/react';
import { FlowStoreType } from '../types/zustand';
import { AllNodeType, EdgeType } from '../types/flow';

export const useFlowStore = create<FlowStoreType>((set, get) => ({
  // Flow state
  nodes: [],
  edges: [],
  componentsToUpdate: [],
  playgroundPage: false,
  lastCopiedSelection: null,
  handleDragging: undefined,
  
  // Node operations
  onNodesChange: (changes) => {
    set({
      nodes: applyNodeChanges(changes, get().nodes) as AllNodeType[],
    });
  },

  setNodes: (update) => {
    const newNodes = typeof update === 'function' ? update(get().nodes) : update;
    set({ nodes: newNodes });
  },

  setNode: (id, update, _isUserChange = true, callback) => {
    const { nodes } = get();
    const nodeIndex = nodes.findIndex((node) => node.id === id);
    
    if (nodeIndex === -1) return;
    
    const updatedNode = typeof update === 'function' 
      ? update(nodes[nodeIndex]) 
      : update;
    
    const newNodes = [...nodes];
    newNodes[nodeIndex] = updatedNode;
    
    set({ nodes: newNodes });
    
    if (callback) callback();
  },

  getNode: (id) => {
    return get().nodes.find((node) => node.id === id);
  },

  deleteNode: (nodeId) => {
    const nodeIds = Array.isArray(nodeId) ? nodeId : [nodeId];
    set({
      nodes: get().nodes.filter((node) => !nodeIds.includes(node.id)),
      // Also remove edges connected to these nodes
      edges: get().edges.filter(
        (edge) => !nodeIds.includes(edge.source) && !nodeIds.includes(edge.target)
      ),
    });
  },

  // Edge operations
  onEdgesChange: (changes) => {
    set({
      edges: applyEdgeChanges(changes, get().edges) as EdgeType[],
    });
  },

  setEdges: (update) => {
    const newEdges = typeof update === 'function' ? update(get().edges) : update;
    set({ edges: newEdges });
  },

  deleteEdge: (edgeId) => {
    const edgeIds = Array.isArray(edgeId) ? edgeId : [edgeId];
    set({
      edges: get().edges.filter((edge) => !edgeIds.includes(edge.id)),
    });
  },

  onConnect: (connection: Connection) => {
    // Create a new edge when a connection is made
    const newEdge: EdgeType = {
      id: `e-${connection.source}-${connection.target}-${Math.floor(Math.random() * 1000)}`,
      source: connection.source || '',
      target: connection.target || '',
      sourceHandle: connection.sourceHandle,
      targetHandle: connection.targetHandle,
      type: 'smoothstep',
      animated: true
    };
    
    set({
      edges: [...get().edges, newEdge],
    });
  },

  // Flow utilities
  cleanFlow: () => {
    set({ nodes: [], edges: [] });
  },

  unselectAll: () => {
    set({
      nodes: get().nodes.map(node => ({ ...node, selected: false })),
      edges: get().edges.map(edge => ({ ...edge, selected: false })),
    });
  },

  getFlow: () => {
    const defaultViewport: Viewport = { x: 0, y: 0, zoom: 1 };
    return { 
      nodes: get().nodes, 
      edges: get().edges, 
      viewport: defaultViewport // Note: you might want to store the actual viewport from React Flow
    };
  },

  getNodePosition: (nodeId) => {
    const node = get().nodes.find(n => n.id === nodeId);
    return node?.position || { x: 0, y: 0 };
  },

  // Handle dragging state for connections
  setHandleDragging: (data) => {
    set({ handleDragging: data });
  },

  // Initialization with default properties
  fitViewNode: () => {
    // Will be replaced when setReactFlowInstance is called
    console.warn('fitViewNode called before ReactFlow instance was set');
  },
}));