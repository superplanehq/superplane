import { AnnotationComponent } from "../../annotationComponent";
import { ComponentBase } from "../../componentBase";
import { Composite } from "../../composite";
import { Trigger } from "../../trigger";
import type { BlockProps, ComponentActionKeys } from "./types";
import {
  buildFallbackComponentProps,
  getSafeAnnotationProps,
  getSafeComponentProps,
  getSafeCompositeProps,
  getSafeTriggerProps,
} from "./data";
import { isRecord } from "@/pages/app/mappers/safeMappers";
import { isVerticalCanvasFlow } from "@/lib/canvasFlowDirection";

function getCompactView(data: BlockProps["data"], isCompactView: BlockProps["isCompactView"]) {
  // Factory nodes always render expanded — collapse chrome is disabled.
  if (isVerticalCanvasFlow(data._flowDirection)) {
    return false;
  }

  if (isCompactView !== undefined) {
    return isCompactView;
  }

  switch (data.type) {
    case "composite":
      return !!data.composite?.collapsed;
    case "trigger":
      return !!data.trigger?.collapsed;
    case "component":
      return !!data.component?.collapsed;
    default:
      return false;
  }
}

function getActionProps(
  compactView: boolean,
  props: Pick<BlockProps, ComponentActionKeys>,
  options?: { omitToggleView?: boolean },
) {
  return {
    onDuplicate: props.onDuplicate,
    onToggleView: options?.omitToggleView ? undefined : props.onToggleView,
    onDelete: props.onDelete,
    isCompactView: compactView,
  };
}

function renderFallbackBlock(args: {
  data: BlockProps["data"];
  fallbackTitle: string;
  selected: boolean;
  showHeader: boolean | undefined;
  canvasMode: BlockProps["canvasMode"];
  showRuntimeStatus?: boolean;
  actionProps: ReturnType<typeof getActionProps>;
  dimBodyBelowHeader?: boolean;
}) {
  const { data, fallbackTitle, selected, showHeader, canvasMode, showRuntimeStatus, actionProps, dimBodyBelowHeader } =
    args;

  return (
    <ComponentBase
      {...buildFallbackComponentProps(data, fallbackTitle)}
      canvasMode={canvasMode}
      showRuntimeStatus={showRuntimeStatus}
      selected={selected}
      showHeader={showHeader}
      dimBodyBelowHeader={dimBodyBelowHeader}
      draftDiffStatus={data._draftDiffStatus}
      isFactoryApp={isVerticalCanvasFlow(data._flowDirection)}
      {...actionProps}
    />
  );
}

function AnnotationBlockContent({
  data,
  nodeId,
  selected,
  showHeader,
  canvasMode,
  showRuntimeStatus,
  onAnnotationUpdate,
  onAnnotationBlur,
  actionProps,
  dimBodyBelowHeader,
}: {
  data: BlockProps["data"];
  nodeId?: string;
  selected: boolean;
  showHeader?: boolean;
  canvasMode?: BlockProps["canvasMode"];
  showRuntimeStatus?: boolean;
  onAnnotationUpdate?: BlockProps["onAnnotationUpdate"];
  onAnnotationBlur?: BlockProps["onAnnotationBlur"];
  actionProps: ReturnType<typeof getActionProps>;
  dimBodyBelowHeader?: boolean;
}) {
  const safeAnnotationProps = getSafeAnnotationProps(data);
  const handleAnnotationUpdate = (updates: {
    text?: string;
    color?: string;
    width?: number;
    height?: number;
    x?: number;
    y?: number;
  }) => {
    if (nodeId && onAnnotationUpdate) {
      onAnnotationUpdate(nodeId, updates);
    }
  };

  if (!safeAnnotationProps) {
    return renderFallbackBlock({
      data,
      fallbackTitle: "Annotation",
      selected,
      showHeader,
      canvasMode,
      showRuntimeStatus,
      actionProps,
      dimBodyBelowHeader,
    });
  }

  return (
    <AnnotationComponent
      {...safeAnnotationProps}
      noteId={nodeId}
      selected={selected}
      canvasMode={canvasMode}
      onAnnotationUpdate={handleAnnotationUpdate}
      onAnnotationBlur={onAnnotationBlur}
      dimBodyBelowHeader={dimBodyBelowHeader}
      draftDiffStatus={data._draftDiffStatus}
      {...actionProps}
    />
  );
}

function renderBlockByType(args: {
  data: BlockProps["data"];
  nodeId?: string;
  selected: boolean;
  showHeader?: boolean;
  canvasMode?: BlockProps["canvasMode"];
  showRuntimeStatus?: boolean;
  onAnnotationUpdate?: BlockProps["onAnnotationUpdate"];
  onAnnotationBlur?: BlockProps["onAnnotationBlur"];
  actionProps: ReturnType<typeof getActionProps>;
  dimBodyBelowHeader?: boolean;
}) {
  const {
    data,
    nodeId,
    selected,
    showHeader,
    canvasMode,
    showRuntimeStatus,
    onAnnotationUpdate,
    onAnnotationBlur,
    actionProps,
    dimBodyBelowHeader,
  } = args;
  const draftDiffStatus = data._draftDiffStatus;

  switch (data.type) {
    case "trigger":
      if (!isRecord(data.trigger)) {
        return renderFallbackBlock({
          data,
          fallbackTitle: "Trigger",
          selected,
          showHeader,
          canvasMode,
          showRuntimeStatus,
          actionProps,
          dimBodyBelowHeader,
        });
      }
      return (
        <Trigger
          {...getSafeTriggerProps(data)}
          canvasMode={canvasMode}
          showRuntimeStatus={showRuntimeStatus}
          selected={selected}
          showHeader={showHeader}
          dimBodyBelowHeader={dimBodyBelowHeader}
          draftDiffStatus={draftDiffStatus}
          isFactoryApp={isVerticalCanvasFlow(data._flowDirection)}
          {...actionProps}
        />
      );
    case "component": {
      const safeComponentProps = getSafeComponentProps(data);
      return (
        <ComponentBase
          {...safeComponentProps}
          canvasMode={canvasMode}
          showRuntimeStatus={showRuntimeStatus}
          selected={selected}
          showHeader={showHeader}
          dimBodyBelowHeader={dimBodyBelowHeader}
          draftDiffStatus={draftDiffStatus}
          isFactoryApp={isVerticalCanvasFlow(data._flowDirection)}
          {...actionProps}
        />
      );
    }
    case "composite":
      return (
        <Composite
          {...getSafeCompositeProps(data)}
          canvasMode={canvasMode}
          showRuntimeStatus={showRuntimeStatus}
          selected={selected}
          showHeader={showHeader}
          dimBodyBelowHeader={dimBodyBelowHeader}
          draftDiffStatus={draftDiffStatus}
          isFactoryApp={isVerticalCanvasFlow(data._flowDirection)}
          {...actionProps}
        />
      );
    case "annotation":
      return (
        <AnnotationBlockContent
          data={data}
          nodeId={nodeId}
          selected={selected}
          showHeader={showHeader}
          canvasMode={canvasMode}
          showRuntimeStatus={showRuntimeStatus}
          onAnnotationUpdate={onAnnotationUpdate}
          onAnnotationBlur={onAnnotationBlur}
          actionProps={actionProps}
          dimBodyBelowHeader={dimBodyBelowHeader}
        />
      );
    default:
      return renderFallbackBlock({
        data,
        fallbackTitle: "Component",
        selected,
        showHeader,
        canvasMode,
        showRuntimeStatus,
        actionProps,
        dimBodyBelowHeader,
      });
  }
}

export function BlockContent({
  data,
  nodeId,
  selected = false,
  onDuplicate,
  onToggleView,
  onDelete,
  showHeader,
  canvasMode,
  showRuntimeStatus,
  isCompactView,
  onAnnotationUpdate,
  onAnnotationBlur,
  dimBodyBelowHeader,
}: BlockProps) {
  const isFactoryApp = isVerticalCanvasFlow(data._flowDirection);
  const compactView = getCompactView(data, isCompactView);
  const actionProps = getActionProps(
    compactView,
    {
      onDuplicate,
      onToggleView,
      onDelete,
    },
    { omitToggleView: isFactoryApp },
  );

  return renderBlockByType({
    data,
    nodeId,
    selected,
    showHeader,
    canvasMode,
    showRuntimeStatus,
    onAnnotationUpdate,
    onAnnotationBlur,
    actionProps,
    dimBodyBelowHeader,
  });
}
