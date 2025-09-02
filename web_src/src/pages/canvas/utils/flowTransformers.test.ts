import { describe, it, expect, vi, beforeEach } from 'vitest';
import { autoLayoutNodes } from './flowTransformers';
import { AllNodeType, EdgeType, StageData } from '../types/flow';
import { ConnectionLineType, MarkerType } from '@xyflow/react';

// Mock ELK layout
vi.mock('./layoutConfig', () => ({
  elk: {
    layout: vi.fn()
  }
}));

// Mock DOM queries
Object.defineProperty(global, 'document', {
  value: {
    querySelector: vi.fn(() => ({ offsetHeight: 100 }))
  }
});

describe('autoLayoutNodes', () => {
  const mockElk = vi.hoisted(() => ({
    layout: vi.fn()
  }));

  beforeEach(() => {
    vi.clearAllMocks();
  });

  const createMockNode = (id: string): AllNodeType => ({
    id,
    type: 'stage',
    data: {
      name: `Node ${id}`,
    } as StageData,
    position: { x: 0, y: 0 }
  });

  const createMockEdge = (id: string, source: string, target: string): EdgeType => ({
    id,
    source,
    target,
    type: ConnectionLineType.Bezier,
    animated: false,
    style: { stroke: '#707070', strokeWidth: 2 },
    markerEnd: { type: MarkerType.ArrowClosed, color: '#707070', strokeWidth: 2 }
  });

  it('should handle normal case with valid nodes and edges', async () => {
    const nodes = [
      createMockNode('node1'),
      createMockNode('node2'),
      createMockNode('node3')
    ];

    const edges = [
      createMockEdge('edge1', 'node1', 'node2'),
      createMockEdge('edge2', 'node2', 'node3')
    ];

    const mockLayoutResult = {
      children: [
        { id: 'node1', x: 100, y: 200 },
        { id: 'node2', x: 300, y: 200 },
        { id: 'node3', x: 500, y: 200 }
      ]
    };

    mockElk.layout.mockResolvedValue(mockLayoutResult);
    
    // Import after mocking
    const { elk } = await import('./layoutConfig');
    vi.mocked(elk.layout).mockResolvedValue(mockLayoutResult);

    const result = await autoLayoutNodes(nodes, edges);

    expect(result).toHaveLength(3);
    expect(result[0].position.x).toBeCloseTo(100, 3);
    expect(result[0].position.y).toBeCloseTo(150, 0);
    expect(result[1].position.x).toBeCloseTo(300, 3);
    expect(result[2].position.x).toBeCloseTo(500, 3);

    // Verify ELK was called with filtered edges (all edges should be valid)
    expect(elk.layout).toHaveBeenCalledWith({
      id: 'root',
      children: expect.arrayContaining([
        { id: 'node1', width: 350, height: 450 },
        { id: 'node2', width: 350, height: 450 },
        { id: 'node3', width: 350, height: 450 }
      ]),
      edges: expect.arrayContaining([
        { id: 'edge1', sources: ['node1'], targets: ['node2'] },
        { id: 'edge2', sources: ['node2'], targets: ['node3'] }
      ])
    });
  });

  it('should filter out edges that reference non-existent nodes', async () => {
    const nodes = [
      createMockNode('node1'),
      createMockNode('node2')
    ];

    const edges = [
      createMockEdge('edge1', 'node1', 'node2'), // Valid edge
      createMockEdge('edge2', 'node1', 'nonexistent'), // Invalid target
      createMockEdge('edge3', 'nonexistent', 'node2'), // Invalid source
      createMockEdge('edge4', 'missing1', 'missing2') // Both missing
    ];

    const mockLayoutResult = {
      children: [
        { id: 'node1', x: 100, y: 200 },
        { id: 'node2', x: 300, y: 200 }
      ]
    };

    mockElk.layout.mockResolvedValue(mockLayoutResult);
    
    // Import after mocking
    const { elk } = await import('./layoutConfig');
    vi.mocked(elk.layout).mockResolvedValue(mockLayoutResult);

    const result = await autoLayoutNodes(nodes, edges);

    expect(result).toHaveLength(2);

    // Verify ELK was called with only the valid edge
    expect(elk.layout).toHaveBeenCalledWith({
      id: 'root',
      children: [
        { id: 'node1', width: 350, height: 450 },
        { id: 'node2', width: 350, height: 450 }
      ],
      edges: [
        { id: 'edge1', sources: ['node1'], targets: ['node2'] }
      ]
    });
  });

  it('should handle ELK layout failure gracefully', async () => {
    const nodes = [createMockNode('node1')];
    const edges = [createMockEdge('edge1', 'node1', 'node2')];

    const consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    
    const { elk } = await import('./layoutConfig');
    vi.mocked(elk.layout).mockRejectedValue(new Error('Layout failed'));

    const result = await autoLayoutNodes(nodes, edges);

    expect(result).toEqual(nodes);
    expect(consoleErrorSpy).toHaveBeenCalledWith('ELK auto-layout failed:', expect.any(Error));
    
    consoleErrorSpy.mockRestore();
  });

  it('should handle empty nodes and edges arrays', async () => {
    const mockLayoutResult = { children: [] };
    
    const { elk } = await import('./layoutConfig');
    vi.mocked(elk.layout).mockResolvedValue(mockLayoutResult);

    const result = await autoLayoutNodes([], []);

    expect(result).toEqual([]);
    expect(elk.layout).toHaveBeenCalledWith({
      id: 'root',
      children: [],
      edges: []
    });
  });

  it('should deduplicate nodes and edges by ID', async () => {
    const nodes = [
      createMockNode('node1'),
      createMockNode('node1'), // Duplicate
      createMockNode('node2')
    ];

    const edges = [
      createMockEdge('edge1', 'node1', 'node2'),
      createMockEdge('edge1', 'node1', 'node2'), // Duplicate
    ];

    const mockLayoutResult = {
      children: [
        { id: 'node1', x: 100, y: 200 },
        { id: 'node2', x: 300, y: 200 }
      ]
    };

    // Import after mocking
    const { elk } = await import('./layoutConfig');
    vi.mocked(elk.layout).mockResolvedValue(mockLayoutResult);

    const result = await autoLayoutNodes(nodes, edges);

    expect(result).toHaveLength(3); // Should return all 3 nodes (deduplication happens in ELK processing, not in result)

    // Verify ELK was called with deduplicated data
    expect(elk.layout).toHaveBeenCalledWith({
      id: 'root',
      children: [
        { id: 'node1', width: 350, height: 450 },
        { id: 'node2', width: 350, height: 450 }
      ],
      edges: [
        { id: 'edge1', sources: ['node1'], targets: ['node2'] }
      ]
    });
  });
});