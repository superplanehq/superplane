import React from 'react';
import type { EdgeProps } from '../../types';
export type GetStraightPathParams = {
    sourceX: number;
    sourceY: number;
    targetX: number;
    targetY: number;
};
export declare function getStraightPath({ sourceX, sourceY, targetX, targetY, }: GetStraightPathParams): [path: string, labelX: number, labelY: number, offsetX: number, offsetY: number];
declare const StraightEdge: React.MemoExoticComponent<({ sourceX, sourceY, targetX, targetY, label, labelStyle, labelShowBg, labelBgStyle, labelBgPadding, labelBgBorderRadius, style, markerEnd, markerStart, interactionWidth, }: EdgeProps) => JSX.Element>;
export default StraightEdge;
//# sourceMappingURL=StraightEdge.d.ts.map