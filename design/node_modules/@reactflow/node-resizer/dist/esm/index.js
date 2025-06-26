import React, { memo, useRef, useEffect } from 'react';
import cc from 'classcat';
import { drag } from 'd3-drag';
import { select } from 'd3-selection';
import { useNodeId, useStoreApi, useGetPointerPosition, clamp } from '@reactflow/core';

var ResizeControlVariant;
(function (ResizeControlVariant) {
    ResizeControlVariant["Line"] = "line";
    ResizeControlVariant["Handle"] = "handle";
})(ResizeControlVariant || (ResizeControlVariant = {}));

// returns an array of two numbers (0, 1 or -1) representing the direction of the resize
// 0 = no change, 1 = increase, -1 = decrease
function getDirection({ width, prevWidth, height, prevHeight, invertX, invertY }) {
    const deltaWidth = width - prevWidth;
    const deltaHeight = height - prevHeight;
    const direction = [deltaWidth > 0 ? 1 : deltaWidth < 0 ? -1 : 0, deltaHeight > 0 ? 1 : deltaHeight < 0 ? -1 : 0];
    if (deltaWidth && invertX) {
        direction[0] = direction[0] * -1;
    }
    if (deltaHeight && invertY) {
        direction[1] = direction[1] * -1;
    }
    return direction;
}

const initPrevValues = { width: 0, height: 0, x: 0, y: 0 };
const initStartValues = {
    ...initPrevValues,
    pointerX: 0,
    pointerY: 0,
    aspectRatio: 1,
};
function ResizeControl({ nodeId, position, variant = ResizeControlVariant.Handle, className, style = {}, children, color, minWidth = 10, minHeight = 10, maxWidth = Number.MAX_VALUE, maxHeight = Number.MAX_VALUE, keepAspectRatio = false, shouldResize, onResizeStart, onResize, onResizeEnd, }) {
    const contextNodeId = useNodeId();
    const id = typeof nodeId === 'string' ? nodeId : contextNodeId;
    const store = useStoreApi();
    const resizeControlRef = useRef(null);
    const startValues = useRef(initStartValues);
    const prevValues = useRef(initPrevValues);
    const getPointerPosition = useGetPointerPosition();
    const defaultPosition = variant === ResizeControlVariant.Line ? 'right' : 'bottom-right';
    const controlPosition = position ?? defaultPosition;
    useEffect(() => {
        if (!resizeControlRef.current || !id) {
            return;
        }
        const selection = select(resizeControlRef.current);
        const enableX = controlPosition.includes('right') || controlPosition.includes('left');
        const enableY = controlPosition.includes('bottom') || controlPosition.includes('top');
        const invertX = controlPosition.includes('left');
        const invertY = controlPosition.includes('top');
        const dragHandler = drag()
            .on('start', (event) => {
            const node = store.getState().nodeInternals.get(id);
            const { xSnapped, ySnapped } = getPointerPosition(event);
            prevValues.current = {
                width: node?.width ?? 0,
                height: node?.height ?? 0,
                x: node?.position.x ?? 0,
                y: node?.position.y ?? 0,
            };
            startValues.current = {
                ...prevValues.current,
                pointerX: xSnapped,
                pointerY: ySnapped,
                aspectRatio: prevValues.current.width / prevValues.current.height,
            };
            onResizeStart?.(event, { ...prevValues.current });
        })
            .on('drag', (event) => {
            const { nodeInternals, triggerNodeChanges } = store.getState();
            const { xSnapped, ySnapped } = getPointerPosition(event);
            const node = nodeInternals.get(id);
            if (node) {
                const changes = [];
                const { pointerX: startX, pointerY: startY, width: startWidth, height: startHeight, x: startNodeX, y: startNodeY, aspectRatio, } = startValues.current;
                const { x: prevX, y: prevY, width: prevWidth, height: prevHeight } = prevValues.current;
                const distX = Math.floor(enableX ? xSnapped - startX : 0);
                const distY = Math.floor(enableY ? ySnapped - startY : 0);
                let width = clamp(startWidth + (invertX ? -distX : distX), minWidth, maxWidth);
                let height = clamp(startHeight + (invertY ? -distY : distY), minHeight, maxHeight);
                if (keepAspectRatio) {
                    const nextAspectRatio = width / height;
                    const isDiagonal = enableX && enableY;
                    const isHorizontal = enableX && !enableY;
                    const isVertical = enableY && !enableX;
                    width = (nextAspectRatio <= aspectRatio && isDiagonal) || isVertical ? height * aspectRatio : width;
                    height = (nextAspectRatio > aspectRatio && isDiagonal) || isHorizontal ? width / aspectRatio : height;
                    if (width >= maxWidth) {
                        width = maxWidth;
                        height = maxWidth / aspectRatio;
                    }
                    else if (width <= minWidth) {
                        width = minWidth;
                        height = minWidth / aspectRatio;
                    }
                    if (height >= maxHeight) {
                        height = maxHeight;
                        width = maxHeight * aspectRatio;
                    }
                    else if (height <= minHeight) {
                        height = minHeight;
                        width = minHeight * aspectRatio;
                    }
                }
                const isWidthChange = width !== prevWidth;
                const isHeightChange = height !== prevHeight;
                if (invertX || invertY) {
                    const x = invertX ? startNodeX - (width - startWidth) : startNodeX;
                    const y = invertY ? startNodeY - (height - startHeight) : startNodeY;
                    // only transform the node if the width or height changes
                    const isXPosChange = x !== prevX && isWidthChange;
                    const isYPosChange = y !== prevY && isHeightChange;
                    if (isXPosChange || isYPosChange) {
                        const positionChange = {
                            id: node.id,
                            type: 'position',
                            position: {
                                x: isXPosChange ? x : prevX,
                                y: isYPosChange ? y : prevY,
                            },
                        };
                        changes.push(positionChange);
                        prevValues.current.x = positionChange.position.x;
                        prevValues.current.y = positionChange.position.y;
                    }
                }
                if (isWidthChange || isHeightChange) {
                    const dimensionChange = {
                        id: id,
                        type: 'dimensions',
                        updateStyle: true,
                        resizing: true,
                        dimensions: {
                            width: width,
                            height: height,
                        },
                    };
                    changes.push(dimensionChange);
                    prevValues.current.width = width;
                    prevValues.current.height = height;
                }
                if (changes.length === 0) {
                    return;
                }
                const direction = getDirection({
                    width: prevValues.current.width,
                    prevWidth,
                    height: prevValues.current.height,
                    prevHeight,
                    invertX,
                    invertY,
                });
                const nextValues = { ...prevValues.current, direction };
                const callResize = shouldResize?.(event, nextValues);
                if (callResize === false) {
                    return;
                }
                onResize?.(event, nextValues);
                triggerNodeChanges(changes);
            }
        })
            .on('end', (event) => {
            const dimensionChange = {
                id: id,
                type: 'dimensions',
                resizing: false,
            };
            onResizeEnd?.(event, { ...prevValues.current });
            store.getState().triggerNodeChanges([dimensionChange]);
        });
        selection.call(dragHandler);
        return () => {
            selection.on('.drag', null);
        };
    }, [
        id,
        controlPosition,
        minWidth,
        minHeight,
        maxWidth,
        maxHeight,
        keepAspectRatio,
        getPointerPosition,
        onResizeStart,
        onResize,
        onResizeEnd,
    ]);
    const positionClassNames = controlPosition.split('-');
    const colorStyleProp = variant === ResizeControlVariant.Line ? 'borderColor' : 'backgroundColor';
    const controlStyle = color ? { ...style, [colorStyleProp]: color } : style;
    return (React.createElement("div", { className: cc(['react-flow__resize-control', 'nodrag', ...positionClassNames, variant, className]), ref: resizeControlRef, style: controlStyle }, children));
}
var ResizeControl$1 = memo(ResizeControl);

const handleControls = ['top-left', 'top-right', 'bottom-left', 'bottom-right'];
const lineControls = ['top', 'right', 'bottom', 'left'];
function NodeResizer({ nodeId, isVisible = true, handleClassName, handleStyle, lineClassName, lineStyle, color, minWidth = 10, minHeight = 10, maxWidth = Number.MAX_VALUE, maxHeight = Number.MAX_VALUE, keepAspectRatio = false, shouldResize, onResizeStart, onResize, onResizeEnd, }) {
    if (!isVisible) {
        return null;
    }
    return (React.createElement(React.Fragment, null,
        lineControls.map((c) => (React.createElement(ResizeControl$1, { key: c, className: lineClassName, style: lineStyle, nodeId: nodeId, position: c, variant: ResizeControlVariant.Line, color: color, minWidth: minWidth, minHeight: minHeight, maxWidth: maxWidth, maxHeight: maxHeight, onResizeStart: onResizeStart, keepAspectRatio: keepAspectRatio, shouldResize: shouldResize, onResize: onResize, onResizeEnd: onResizeEnd }))),
        handleControls.map((c) => (React.createElement(ResizeControl$1, { key: c, className: handleClassName, style: handleStyle, nodeId: nodeId, position: c, color: color, minWidth: minWidth, minHeight: minHeight, maxWidth: maxWidth, maxHeight: maxHeight, onResizeStart: onResizeStart, keepAspectRatio: keepAspectRatio, shouldResize: shouldResize, onResize: onResize, onResizeEnd: onResizeEnd })))));
}

export { ResizeControl$1 as NodeResizeControl, NodeResizer, ResizeControlVariant };
