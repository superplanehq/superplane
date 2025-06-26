import type { SetStateAction, Dispatch } from 'react';
import type { Node, NodeChange, Edge, EdgeChange } from '../types';
type OnChange<ChangesType> = (changes: ChangesType[]) => void;
export declare const useNodesState: <NodeData = any>(initialItems: Node<NodeData, string | undefined>[]) => [Node<NodeData, string | undefined>[], Dispatch<SetStateAction<Node<NodeData, string | undefined>[]>>, OnChange<NodeChange>];
export declare const useEdgesState: <EdgeData = any>(initialItems: Edge<EdgeData>[]) => [Edge<EdgeData>[], Dispatch<SetStateAction<Edge<EdgeData>[]>>, OnChange<EdgeChange>];
export {};
//# sourceMappingURL=useNodesEdgesState.d.ts.map