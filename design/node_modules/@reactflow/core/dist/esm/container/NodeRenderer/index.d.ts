import React from 'react';
import { GraphViewProps } from '../GraphView';
import type { NodeTypesWrapped } from '../../types';
type NodeRendererProps = Pick<GraphViewProps, 'selectNodesOnDrag' | 'onNodeClick' | 'onNodeDoubleClick' | 'onNodeMouseEnter' | 'onNodeMouseMove' | 'onNodeMouseLeave' | 'onNodeContextMenu' | 'onlyRenderVisibleElements' | 'noPanClassName' | 'noDragClassName' | 'rfId' | 'disableKeyboardA11y' | 'nodeOrigin' | 'nodeExtent'> & {
    nodeTypes: NodeTypesWrapped;
};
declare const _default: React.MemoExoticComponent<{
    (props: NodeRendererProps): JSX.Element;
    displayName: string;
}>;
export default _default;
//# sourceMappingURL=index.d.ts.map