import React, { useCallback } from 'react';
import { useStore, useNodeId, getNodesBounds, internalsSymbol, Position } from '@reactflow/core';
import cc from 'classcat';
import { shallow } from 'zustand/shallow';
import { createPortal } from 'react-dom';

const selector = (state) => state.domNode?.querySelector('.react-flow__renderer');
function NodeToolbarPortal({ children }) {
    const wrapperRef = useStore(selector);
    if (!wrapperRef) {
        return null;
    }
    return createPortal(children, wrapperRef);
}

const nodeEqualityFn = (a, b) => a?.positionAbsolute?.x === b?.positionAbsolute?.x &&
    a?.positionAbsolute?.y === b?.positionAbsolute?.y &&
    a?.width === b?.width &&
    a?.height === b?.height &&
    a?.selected === b?.selected &&
    a?.[internalsSymbol]?.z === b?.[internalsSymbol]?.z;
const nodesEqualityFn = (a, b) => {
    return a.length === b.length && a.every((node, i) => nodeEqualityFn(node, b[i]));
};
const storeSelector = (state) => ({
    transform: state.transform,
    nodeOrigin: state.nodeOrigin,
    selectedNodesCount: state.getNodes().filter((node) => node.selected).length,
});
function getTransform(nodeRect, transform, position, offset, align) {
    let alignmentOffset = 0.5;
    if (align === 'start') {
        alignmentOffset = 0;
    }
    else if (align === 'end') {
        alignmentOffset = 1;
    }
    // position === Position.Top
    // we set the x any y position of the toolbar based on the nodes position
    let pos = [
        (nodeRect.x + nodeRect.width * alignmentOffset) * transform[2] + transform[0],
        nodeRect.y * transform[2] + transform[1] - offset,
    ];
    // and than shift it based on the alignment. The shift values are in %.
    let shift = [-100 * alignmentOffset, -100];
    switch (position) {
        case Position.Right:
            pos = [
                (nodeRect.x + nodeRect.width) * transform[2] + transform[0] + offset,
                (nodeRect.y + nodeRect.height * alignmentOffset) * transform[2] + transform[1],
            ];
            shift = [0, -100 * alignmentOffset];
            break;
        case Position.Bottom:
            pos[1] = (nodeRect.y + nodeRect.height) * transform[2] + transform[1] + offset;
            shift[1] = 0;
            break;
        case Position.Left:
            pos = [
                nodeRect.x * transform[2] + transform[0] - offset,
                (nodeRect.y + nodeRect.height * alignmentOffset) * transform[2] + transform[1],
            ];
            shift = [-100, -100 * alignmentOffset];
            break;
    }
    return `translate(${pos[0]}px, ${pos[1]}px) translate(${shift[0]}%, ${shift[1]}%)`;
}
function NodeToolbar({ nodeId, children, className, style, isVisible, position = Position.Top, offset = 10, align = 'center', ...rest }) {
    const contextNodeId = useNodeId();
    const nodesSelector = useCallback((state) => {
        const nodeIds = Array.isArray(nodeId) ? nodeId : [nodeId || contextNodeId || ''];
        return nodeIds.reduce((acc, id) => {
            const node = state.nodeInternals.get(id);
            if (node) {
                acc.push(node);
            }
            return acc;
        }, []);
    }, [nodeId, contextNodeId]);
    const nodes = useStore(nodesSelector, nodesEqualityFn);
    const { transform, nodeOrigin, selectedNodesCount } = useStore(storeSelector, shallow);
    const isActive = typeof isVisible === 'boolean' ? isVisible : nodes.length === 1 && nodes[0].selected && selectedNodesCount === 1;
    if (!isActive || !nodes.length) {
        return null;
    }
    const nodeRect = getNodesBounds(nodes, nodeOrigin);
    const zIndex = Math.max(...nodes.map((node) => (node[internalsSymbol]?.z || 1) + 1));
    const wrapperStyle = {
        position: 'absolute',
        transform: getTransform(nodeRect, transform, position, offset, align),
        zIndex,
        ...style,
    };
    return (React.createElement(NodeToolbarPortal, null,
        React.createElement("div", { style: wrapperStyle, className: cc(['react-flow__node-toolbar', className]), ...rest }, children)));
}

export { NodeToolbar };
