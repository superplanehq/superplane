import React, { ReactNode } from 'react';
import { GraphViewProps } from '../GraphView';
import type { EdgeTypesWrapped } from '../../types';
type EdgeRendererProps = Pick<GraphViewProps, 'onEdgeClick' | 'onEdgeDoubleClick' | 'defaultMarkerColor' | 'onlyRenderVisibleElements' | 'onEdgeContextMenu' | 'onEdgeMouseEnter' | 'onEdgeMouseMove' | 'onEdgeMouseLeave' | 'onReconnect' | 'onReconnectStart' | 'onReconnectEnd' | 'reconnectRadius' | 'noPanClassName' | 'elevateEdgesOnSelect' | 'rfId' | 'disableKeyboardA11y'> & {
    edgeTypes: EdgeTypesWrapped;
    elevateEdgesOnSelect: boolean;
    children: ReactNode;
};
declare const _default: React.MemoExoticComponent<{
    ({ defaultMarkerColor, onlyRenderVisibleElements, elevateEdgesOnSelect, rfId, edgeTypes, noPanClassName, onEdgeContextMenu, onEdgeMouseEnter, onEdgeMouseMove, onEdgeMouseLeave, onEdgeClick, onEdgeDoubleClick, onReconnect, onReconnectStart, onReconnectEnd, reconnectRadius, children, disableKeyboardA11y, }: EdgeRendererProps): JSX.Element | null;
    displayName: string;
}>;
export default _default;
//# sourceMappingURL=index.d.ts.map