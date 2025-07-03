import { type NodeDragItem, type XYPosition, InternalNodeBase, NodeBase, NodeLookup } from '../types';
export declare function isParentSelected<NodeType extends NodeBase>(node: NodeType, nodeLookup: NodeLookup): boolean;
export declare function hasSelector(target: Element | EventTarget | null, selector: string, domNode: Element): boolean;
export declare function getDragItems<NodeType extends NodeBase>(nodeLookup: Map<string, InternalNodeBase<NodeType>>, nodesDraggable: boolean, mousePos: XYPosition, nodeId?: string): Map<string, NodeDragItem>;
export declare function getEventHandlerParams<NodeType extends InternalNodeBase>({ nodeId, dragItems, nodeLookup, dragging, }: {
    nodeId?: string;
    dragItems: Map<string, NodeDragItem>;
    nodeLookup: Map<string, NodeType>;
    dragging?: boolean;
}): [NodeBase, NodeBase[]];
//# sourceMappingURL=utils.d.ts.map