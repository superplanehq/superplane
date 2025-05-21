import {OnNodesChange, OnEdgesChange, Connection, Viewport} from "@xyflow/react";
import { AllNodeType, EdgeType } from "../flow";


export type FlowStoreType = {
    fitViewNode: (nodeId: string) => void;
    componentsToUpdate: string[];
    nodes: AllNodeType[];
    edges: EdgeType[];
    onNodesChange: OnNodesChange<AllNodeType>;
    onEdgesChange: OnEdgesChange<EdgeType>;
    setNodes: (
      update: AllNodeType[] | ((oldState: AllNodeType[]) => AllNodeType[]),
    ) => void;
    setEdges: (
      update: EdgeType[] | ((oldState: EdgeType[]) => EdgeType[]),
    ) => void;
    setNode: (
      id: string,
      update: AllNodeType | ((oldState: AllNodeType) => AllNodeType),
      isUserChange?: boolean,
      callback?: () => void,
    ) => void;
    getNode: (id: string) => AllNodeType | undefined;
    deleteNode: (nodeId: string | Array<string>) => void;
    deleteEdge: (edgeId: string | Array<string>) => void;
    cleanFlow: () => void;
    onConnect: (connection: Connection) => void;
    unselectAll: () => void;
    playgroundPage: boolean;
    getFlow: () => { nodes: AllNodeType[]; edges: EdgeType[]; viewport: Viewport };
    getNodePosition: (nodeId: string) => { x: number; y: number };
    handleDragging:
      | {
          source: string | undefined;
          sourceHandle: string | undefined;
          target: string | undefined;
          targetHandle: string | undefined;
          type: string;
          color: string;
        }
      | undefined;
    setHandleDragging: (
      data:
        | {
            source: string | undefined;
            sourceHandle: string | undefined;
            target: string | undefined;
            targetHandle: string | undefined;
            type: string;
            color: string;
          }
        | undefined,
    ) => void;
  };