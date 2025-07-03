/**
 * The user selection rectangle gets displayed when a user drags the mouse while pressing shift
 */
import { type ReactNode } from 'react';
import type { ReactFlowProps } from '../../types';
type PaneProps = {
    isSelecting: boolean;
    selectionKeyPressed: boolean;
    children: ReactNode;
} & Partial<Pick<ReactFlowProps, 'selectionMode' | 'panOnDrag' | 'onSelectionStart' | 'onSelectionEnd' | 'onPaneClick' | 'onPaneContextMenu' | 'onPaneScroll' | 'onPaneMouseEnter' | 'onPaneMouseMove' | 'onPaneMouseLeave' | 'selectionOnDrag'>>;
export declare function Pane({ isSelecting, selectionKeyPressed, selectionMode, panOnDrag, selectionOnDrag, onSelectionStart, onSelectionEnd, onPaneClick, onPaneContextMenu, onPaneScroll, onPaneMouseEnter, onPaneMouseMove, onPaneMouseLeave, children, }: PaneProps): import("react/jsx-runtime").JSX.Element;
export {};
//# sourceMappingURL=index.d.ts.map