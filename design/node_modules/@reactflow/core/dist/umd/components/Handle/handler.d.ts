import type { MouseEvent as ReactMouseEvent, TouchEvent as ReactTouchEvent } from 'react';
import { StoreApi } from 'zustand';
import type { OnConnect, HandleType, ReactFlowState } from '../../types';
import { ValidConnectionFunc } from './utils';
export declare function handlePointerDown({ event, handleId, nodeId, onConnect, isTarget, getState, setState, isValidConnection, edgeUpdaterType, onReconnectEnd, }: {
    event: ReactMouseEvent | ReactTouchEvent;
    handleId: string | null;
    nodeId: string;
    onConnect: OnConnect;
    isTarget: boolean;
    getState: StoreApi<ReactFlowState>['getState'];
    setState: StoreApi<ReactFlowState>['setState'];
    isValidConnection: ValidConnectionFunc;
    edgeUpdaterType?: HandleType;
    onReconnectEnd?: (evt: MouseEvent | TouchEvent) => void;
}): void;
//# sourceMappingURL=handler.d.ts.map