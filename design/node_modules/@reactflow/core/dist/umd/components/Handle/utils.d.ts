import { MouseEvent as ReactMouseEvent, TouchEvent as ReactTouchEvent } from 'react';
import { ConnectingHandle, ConnectionMode, ConnectionStatus } from '../../types';
import type { Connection, HandleType, XYPosition, Node, NodeHandleBounds } from '../../types';
export type ConnectionHandle = {
    id: string | null;
    type: HandleType | null;
    nodeId: string;
    x: number;
    y: number;
};
export type ValidConnectionFunc = (connection: Connection) => boolean;
export declare function getHandles(node: Node, handleBounds: NodeHandleBounds, type: HandleType, currentHandle: string): ConnectionHandle[];
export declare function getClosestHandle(event: MouseEvent | TouchEvent | ReactMouseEvent | ReactTouchEvent, doc: Document | ShadowRoot, pos: XYPosition, connectionRadius: number, handles: ConnectionHandle[], validator: (handle: Pick<ConnectionHandle, 'nodeId' | 'id' | 'type'>) => Result): {
    handle: {
        id: string | null;
        type: HandleType | null;
        nodeId: string;
        x: number;
        y: number;
    };
    validHandleResult: Result;
} | {
    handle: null;
    validHandleResult: Result;
};
type Result = {
    handleDomNode: Element | null;
    isValid: boolean;
    connection: Connection;
    endHandle: ConnectingHandle | null;
};
export declare function isValidHandle(handle: Pick<ConnectionHandle, 'nodeId' | 'id' | 'type'>, connectionMode: ConnectionMode, fromNodeId: string, fromHandleId: string | null, fromType: HandleType, isValidConnection: ValidConnectionFunc, doc: Document | ShadowRoot): {
    handleDomNode: Element | null;
    isValid: boolean;
    connection: Connection;
    endHandle: ConnectingHandle | null;
};
type GetHandleLookupParams = {
    nodes: Node[];
    nodeId: string;
    handleId: string | null;
    handleType: string;
};
export declare function getHandleLookup({ nodes, nodeId, handleId, handleType }: GetHandleLookupParams): ConnectionHandle[];
export declare function getHandleType(edgeUpdaterType: HandleType | undefined, handleDomNode: Element | null): HandleType | null;
export declare function resetRecentHandle(handleDomNode: Element): void;
export declare function getConnectionStatus(isInsideConnectionRadius: boolean, isHandleValid: boolean): ConnectionStatus;
export {};
//# sourceMappingURL=utils.d.ts.map