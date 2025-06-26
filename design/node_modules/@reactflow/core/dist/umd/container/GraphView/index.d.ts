import React from 'react';
import type { ReactFlowProps } from '../../types';
export type GraphViewProps = Omit<ReactFlowProps, 'onSelectionChange' | 'nodes' | 'edges' | 'nodeTypes' | 'edgeTypes' | 'onEdgeUpdate' | 'onEdgeUpdateStart' | 'onEdgeUpdateEnd' | 'edgeUpdaterRadius'> & Required<Pick<ReactFlowProps, 'nodeTypes' | 'edgeTypes' | 'selectionKeyCode' | 'deleteKeyCode' | 'multiSelectionKeyCode' | 'connectionLineType' | 'onlyRenderVisibleElements' | 'translateExtent' | 'minZoom' | 'maxZoom' | 'defaultMarkerColor' | 'selectNodesOnDrag' | 'noDragClassName' | 'noDragClassName' | 'noWheelClassName' | 'noPanClassName' | 'defaultViewport' | 'disableKeyboardA11y' | 'nodeOrigin'>> & {
    rfId: string;
};
declare const _default: React.MemoExoticComponent<{
    ({ nodeTypes, edgeTypes, onMove, onMoveStart, onMoveEnd, onInit, onNodeClick, onEdgeClick, onNodeDoubleClick, onEdgeDoubleClick, onNodeMouseEnter, onNodeMouseMove, onNodeMouseLeave, onNodeContextMenu, onSelectionContextMenu, onSelectionStart, onSelectionEnd, connectionLineType, connectionLineStyle, connectionLineComponent, connectionLineContainerStyle, selectionKeyCode, selectionOnDrag, selectionMode, multiSelectionKeyCode, panActivationKeyCode, zoomActivationKeyCode, deleteKeyCode, onlyRenderVisibleElements, elementsSelectable, selectNodesOnDrag, defaultViewport, translateExtent, minZoom, maxZoom, preventScrolling, defaultMarkerColor, zoomOnScroll, zoomOnPinch, panOnScroll, panOnScrollSpeed, panOnScrollMode, zoomOnDoubleClick, panOnDrag, onPaneClick, onPaneMouseEnter, onPaneMouseMove, onPaneMouseLeave, onPaneScroll, onPaneContextMenu, onEdgeContextMenu, onEdgeMouseEnter, onEdgeMouseMove, onEdgeMouseLeave, onReconnect, onReconnectStart, onReconnectEnd, reconnectRadius, noDragClassName, noWheelClassName, noPanClassName, elevateEdgesOnSelect, disableKeyboardA11y, nodeOrigin, nodeExtent, rfId, }: GraphViewProps): JSX.Element;
    displayName: string;
}>;
export default _default;
//# sourceMappingURL=index.d.ts.map