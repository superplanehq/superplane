import { AnnotationComponent } from "../../annotationComponent";
import { ComponentBase } from "../../componentBase";
import { Composite } from "../../composite";
import { Trigger } from "../../trigger";
import type { BlockProps } from "./types";
import {
  buildFallbackComponentProps,
  getSafeAnnotationProps,
  getSafeComponentProps,
  getSafeCompositeProps,
  getSafeTriggerProps,
  isRecord,
} from "./data";

// eslint-disable-next-line max-lines-per-function
export function BlockContent({
  data,
  onExpand,
  nodeId,
  selected = false,
  onAnnotationUpdate,
  onAnnotationBlur,
  onRun,
  runDisabled,
  runDisabledTooltip,
  onTogglePause,
  onEdit,
  onConfigure,
  onDuplicate,
  onDeactivate,
  onToggleCollapse,
  onToggleView,
  onDelete,
  showHeader,
  isCompactView,
}: BlockProps) {
  const compactView =
    isCompactView ??
    (() => {
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
    })();

  const handleExpand = () => {
    if (onExpand && nodeId) {
      onExpand(nodeId, data);
    }
  };

  const actionProps = {
    onRun,
    runDisabled,
    runDisabledTooltip,
    onTogglePause: data.type === "trigger" ? undefined : onTogglePause,
    onEdit,
    onDuplicate,
    onDeactivate,
    onToggleCollapse,
    onToggleView,
    onDelete,
    isCompactView: compactView,
    onConfigure: data.type === "composite" ? onConfigure : undefined,
  };

  switch (data.type) {
    case "trigger":
      if (!isRecord(data.trigger)) {
        return (
          <ComponentBase
            {...buildFallbackComponentProps(data, "Trigger")}
            selected={selected}
            showHeader={showHeader}
            {...actionProps}
          />
        );
      }
      return <Trigger {...getSafeTriggerProps(data)} selected={selected} showHeader={showHeader} {...actionProps} />;
    case "component": {
      const safeComponentProps = getSafeComponentProps(data);
      return (
        <ComponentBase
          {...safeComponentProps}
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
          onExpandChildEvents={handleExpand}
          selected={selected}
          showHeader={showHeader}
          {...actionProps}
        />
      );
    case "annotation": {
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
        return (
          <ComponentBase
            {...buildFallbackComponentProps(data, "Annotation")}
            selected={selected}
            showHeader={showHeader}
            {...actionProps}
          />
        );
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
    case "group":
      return (
        <ComponentBase
          {...buildFallbackComponentProps(data, "Group")}
          selected={selected}
          showHeader={showHeader}
          {...actionProps}
        />
      );
    default:
      return (
        <ComponentBase
          {...buildFallbackComponentProps(data, "Component")}
          selected={selected}
          showHeader={showHeader}
          {...actionProps}
        />
      );
  }
}
