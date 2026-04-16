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
import { isRecord } from "@/pages/workflowv2/mappers/safeMappers";

function getCompactView(data: BlockProps["data"], isCompactView: BlockProps["isCompactView"]) {
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

function getActionProps(data: BlockProps["data"], compactView: boolean, props: Pick<BlockProps, ComponentActionKeys>) {
  return {
    onRun: props.onRun,
    runDisabled: props.runDisabled,
    runDisabledTooltip: props.runDisabledTooltip,
    onTogglePause: data.type === "trigger" ? undefined : props.onTogglePause,
    onEdit: props.onEdit,
    onDuplicate: props.onDuplicate,
    onDeactivate: props.onDeactivate,
    onToggleView: props.onToggleView,
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
  actionProps: ReturnType<typeof getActionProps>;
}) {
  const { data, fallbackTitle, selected, showHeader, canvasMode, actionProps } = args;

  return (
    <ComponentBase
      {...buildFallbackComponentProps(data, fallbackTitle)}
      canvasMode={canvasMode}
      selected={selected}
      showHeader={showHeader}
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
  onAnnotationUpdate,
  onAnnotationBlur,
  actionProps,
}: {
  data: BlockProps["data"];
  nodeId?: string;
  selected: boolean;
  showHeader?: boolean;
  canvasMode?: BlockProps["canvasMode"];
  onAnnotationUpdate?: BlockProps["onAnnotationUpdate"];
  onAnnotationBlur?: BlockProps["onAnnotationBlur"];
  actionProps: ReturnType<typeof getActionProps>;
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
      actionProps,
    });
  }

  return (
    <AnnotationComponent
      {...safeAnnotationProps}
      noteId={nodeId}
      selected={selected}
      onAnnotationUpdate={handleAnnotationUpdate}
      onAnnotationBlur={onAnnotationBlur}
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
  onAnnotationUpdate?: BlockProps["onAnnotationUpdate"];
  onAnnotationBlur?: BlockProps["onAnnotationBlur"];
  actionProps: ReturnType<typeof getActionProps>;
}) {
  const { data, nodeId, selected, showHeader, canvasMode, onAnnotationUpdate, onAnnotationBlur, actionProps } = args;

  switch (data.type) {
    case "trigger":
      if (!isRecord(data.trigger)) {
        return renderFallbackBlock({
          data,
          fallbackTitle: "Trigger",
          selected,
          showHeader,
          canvasMode,
          actionProps,
        });
      }
      return (
        <Trigger
          {...getSafeTriggerProps(data)}
          canvasMode={canvasMode}
          selected={selected}
          showHeader={showHeader}
          {...actionProps}
        />
      );
    case "component": {
      const safeComponentProps = getSafeComponentProps(data);
      return (
        <ComponentBase
          {...safeComponentProps}
          canvasMode={canvasMode}
          paused={safeComponentProps.paused}
          selected={selected}
          showHeader={showHeader}
          {...actionProps}
        />
      );
    }
    case "composite":
      return (
        <Composite
          {...getSafeCompositeProps(data)}
          canvasMode={canvasMode}
          selected={selected}
          showHeader={showHeader}
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
          onAnnotationUpdate={onAnnotationUpdate}
          onAnnotationBlur={onAnnotationBlur}
          actionProps={actionProps}
        />
      );
    case "group":
      return renderFallbackBlock({
        data,
        fallbackTitle: "Group",
        selected,
        showHeader,
        canvasMode,
        actionProps,
      });
    default:
      return renderFallbackBlock({
        data,
        fallbackTitle: "Component",
        selected,
        showHeader,
        canvasMode,
        actionProps,
      });
  }
}

export function BlockContent({
  data,
  nodeId,
  selected = false,
  onRun,
  runDisabled,
  runDisabledTooltip,
  onTogglePause,
  onEdit,
  onDuplicate,
  onDeactivate,
  onToggleView,
  onDelete,
  showHeader,
  canvasMode,
  isCompactView,
  onAnnotationUpdate,
  onAnnotationBlur,
}: BlockProps) {
  const compactView = getCompactView(data, isCompactView);
  const actionProps = getActionProps(data, compactView, {
    onRun,
    runDisabled,
    runDisabledTooltip,
    onTogglePause,
    onEdit,
    onDuplicate,
    onDeactivate,
    onToggleView,
    onDelete,
  });

  return renderBlockByType({
    data,
    nodeId,
    selected,
    showHeader,
    canvasMode,
    onAnnotationUpdate,
    onAnnotationBlur,
    actionProps,
  });
}
