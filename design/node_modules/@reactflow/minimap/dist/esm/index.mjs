import React, { memo, useRef, useEffect } from 'react';
import cc from 'classcat';
import { shallow } from 'zustand/shallow';
import { zoom, zoomIdentity } from 'd3-zoom';
import { select, pointer } from 'd3-selection';
import { useStore, getNodePositionWithOrigin, useStoreApi, Panel, getBoundsOfRects, getNodesBounds } from '@reactflow/core';

const MiniMapNode = ({ id, x, y, width, height, style, color, strokeColor, strokeWidth, className, borderRadius, shapeRendering, onClick, selected, }) => {
    const { background, backgroundColor } = style || {};
    const fill = (color || background || backgroundColor);
    return (React.createElement("rect", { className: cc(['react-flow__minimap-node', { selected }, className]), x: x, y: y, rx: borderRadius, ry: borderRadius, width: width, height: height, fill: fill, stroke: strokeColor, strokeWidth: strokeWidth, shapeRendering: shapeRendering, onClick: onClick ? (event) => onClick(event, id) : undefined }));
};
MiniMapNode.displayName = 'MiniMapNode';
var MiniMapNode$1 = memo(MiniMapNode);

/* eslint-disable @typescript-eslint/ban-ts-comment */
const selector$1 = (s) => s.nodeOrigin;
const selectorNodes = (s) => s.getNodes().filter((node) => !node.hidden && node.width && node.height);
const getAttrFunction = (func) => (func instanceof Function ? func : () => func);
function MiniMapNodes({ nodeStrokeColor = 'transparent', nodeColor = '#e2e2e2', nodeClassName = '', nodeBorderRadius = 5, nodeStrokeWidth = 2, 
// We need to rename the prop to be `CapitalCase` so that JSX will render it as
// a component properly.
nodeComponent: NodeComponent = MiniMapNode$1, onClick, }) {
    const nodes = useStore(selectorNodes, shallow);
    const nodeOrigin = useStore(selector$1);
    const nodeColorFunc = getAttrFunction(nodeColor);
    const nodeStrokeColorFunc = getAttrFunction(nodeStrokeColor);
    const nodeClassNameFunc = getAttrFunction(nodeClassName);
    const shapeRendering = typeof window === 'undefined' || !!window.chrome ? 'crispEdges' : 'geometricPrecision';
    return (React.createElement(React.Fragment, null, nodes.map((node) => {
        const { x, y } = getNodePositionWithOrigin(node, nodeOrigin).positionAbsolute;
        return (React.createElement(NodeComponent, { key: node.id, x: x, y: y, width: node.width, height: node.height, style: node.style, selected: node.selected, className: nodeClassNameFunc(node), color: nodeColorFunc(node), borderRadius: nodeBorderRadius, strokeColor: nodeStrokeColorFunc(node), strokeWidth: nodeStrokeWidth, shapeRendering: shapeRendering, onClick: onClick, id: node.id }));
    })));
}
var MiniMapNodes$1 = memo(MiniMapNodes);

/* eslint-disable @typescript-eslint/ban-ts-comment */
const defaultWidth = 200;
const defaultHeight = 150;
const selector = (s) => {
    const nodes = s.getNodes();
    const viewBB = {
        x: -s.transform[0] / s.transform[2],
        y: -s.transform[1] / s.transform[2],
        width: s.width / s.transform[2],
        height: s.height / s.transform[2],
    };
    return {
        viewBB,
        boundingRect: nodes.length > 0 ? getBoundsOfRects(getNodesBounds(nodes, s.nodeOrigin), viewBB) : viewBB,
        rfId: s.rfId,
    };
};
const ARIA_LABEL_KEY = 'react-flow__minimap-desc';
function MiniMap({ style, className, nodeStrokeColor = 'transparent', nodeColor = '#e2e2e2', nodeClassName = '', nodeBorderRadius = 5, nodeStrokeWidth = 2, 
// We need to rename the prop to be `CapitalCase` so that JSX will render it as
// a component properly.
nodeComponent, maskColor = 'rgb(240, 240, 240, 0.6)', maskStrokeColor = 'none', maskStrokeWidth = 1, position = 'bottom-right', onClick, onNodeClick, pannable = false, zoomable = false, ariaLabel = 'React Flow mini map', inversePan = false, zoomStep = 10, offsetScale = 5, }) {
    const store = useStoreApi();
    const svg = useRef(null);
    const { boundingRect, viewBB, rfId } = useStore(selector, shallow);
    const elementWidth = style?.width ?? defaultWidth;
    const elementHeight = style?.height ?? defaultHeight;
    const scaledWidth = boundingRect.width / elementWidth;
    const scaledHeight = boundingRect.height / elementHeight;
    const viewScale = Math.max(scaledWidth, scaledHeight);
    const viewWidth = viewScale * elementWidth;
    const viewHeight = viewScale * elementHeight;
    const offset = offsetScale * viewScale;
    const x = boundingRect.x - (viewWidth - boundingRect.width) / 2 - offset;
    const y = boundingRect.y - (viewHeight - boundingRect.height) / 2 - offset;
    const width = viewWidth + offset * 2;
    const height = viewHeight + offset * 2;
    const labelledBy = `${ARIA_LABEL_KEY}-${rfId}`;
    const viewScaleRef = useRef(0);
    viewScaleRef.current = viewScale;
    useEffect(() => {
        if (svg.current) {
            const selection = select(svg.current);
            const zoomHandler = (event) => {
                const { transform, d3Selection, d3Zoom } = store.getState();
                if (event.sourceEvent.type !== 'wheel' || !d3Selection || !d3Zoom) {
                    return;
                }
                const pinchDelta = -event.sourceEvent.deltaY *
                    (event.sourceEvent.deltaMode === 1 ? 0.05 : event.sourceEvent.deltaMode ? 1 : 0.002) *
                    zoomStep;
                const zoom = transform[2] * Math.pow(2, pinchDelta);
                d3Zoom.scaleTo(d3Selection, zoom);
            };
            const panHandler = (event) => {
                const { transform, d3Selection, d3Zoom, translateExtent, width, height } = store.getState();
                if (event.sourceEvent.type !== 'mousemove' || !d3Selection || !d3Zoom) {
                    return;
                }
                // @TODO: how to calculate the correct next position? Math.max(1, transform[2]) is a workaround.
                const moveScale = viewScaleRef.current * Math.max(1, transform[2]) * (inversePan ? -1 : 1);
                const position = {
                    x: transform[0] - event.sourceEvent.movementX * moveScale,
                    y: transform[1] - event.sourceEvent.movementY * moveScale,
                };
                const extent = [
                    [0, 0],
                    [width, height],
                ];
                const nextTransform = zoomIdentity.translate(position.x, position.y).scale(transform[2]);
                const constrainedTransform = d3Zoom.constrain()(nextTransform, extent, translateExtent);
                d3Zoom.transform(d3Selection, constrainedTransform);
            };
            const zoomAndPanHandler = zoom()
                // @ts-ignore
                .on('zoom', pannable ? panHandler : null)
                // @ts-ignore
                .on('zoom.wheel', zoomable ? zoomHandler : null);
            selection.call(zoomAndPanHandler);
            return () => {
                selection.on('zoom', null);
            };
        }
    }, [pannable, zoomable, inversePan, zoomStep]);
    const onSvgClick = onClick
        ? (event) => {
            const rfCoord = pointer(event);
            onClick(event, { x: rfCoord[0], y: rfCoord[1] });
        }
        : undefined;
    const onSvgNodeClick = onNodeClick
        ? (event, nodeId) => {
            const node = store.getState().nodeInternals.get(nodeId);
            onNodeClick(event, node);
        }
        : undefined;
    return (React.createElement(Panel, { position: position, style: style, className: cc(['react-flow__minimap', className]), "data-testid": "rf__minimap" },
        React.createElement("svg", { width: elementWidth, height: elementHeight, viewBox: `${x} ${y} ${width} ${height}`, role: "img", "aria-labelledby": labelledBy, ref: svg, onClick: onSvgClick },
            ariaLabel && React.createElement("title", { id: labelledBy }, ariaLabel),
            React.createElement(MiniMapNodes$1, { onClick: onSvgNodeClick, nodeColor: nodeColor, nodeStrokeColor: nodeStrokeColor, nodeBorderRadius: nodeBorderRadius, nodeClassName: nodeClassName, nodeStrokeWidth: nodeStrokeWidth, nodeComponent: nodeComponent }),
            React.createElement("path", { className: "react-flow__minimap-mask", d: `M${x - offset},${y - offset}h${width + offset * 2}v${height + offset * 2}h${-width - offset * 2}z
        M${viewBB.x},${viewBB.y}h${viewBB.width}v${viewBB.height}h${-viewBB.width}z`, fill: maskColor, fillRule: "evenodd", stroke: maskStrokeColor, strokeWidth: maskStrokeWidth, pointerEvents: "none" }))));
}
MiniMap.displayName = 'MiniMap';
var MiniMap$1 = memo(MiniMap);

export { MiniMap$1 as MiniMap };
