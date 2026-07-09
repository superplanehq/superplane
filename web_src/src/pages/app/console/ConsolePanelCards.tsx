import type { ConsolePanel } from "@/hooks/useCanvasData";

import { ChartPanelCard } from "./ChartPanelCard";
import { HtmlPanelCard } from "./HtmlPanelCard";
import { MarkdownPanelCard } from "./MarkdownPanelCard";
import { NodesPanelCard } from "./NodesPanelCard";
import { NumberPanelCard } from "./NumberPanelCard";
import { TablePanelCard } from "./TablePanelCard";

export function PanelCardRouter({
  panel,
  readOnly,
  onDelete,
  onChange,
  onEditingChange,
}: {
  panel: ConsolePanel;
  readOnly: boolean;
  onDelete: () => void;
  onChange: (content: Record<string, unknown>) => void;
  onEditingChange?: (editing: boolean) => void;
}) {
  switch (panel.type) {
    case "node":
    case "nodes":
      // Both the legacy `node` shape and the modern `nodes` list are rendered
      // by the same merged card — it folds a single-entry list into the
      // compact centered layout that the pre-merge single-node card used.
      return (
        <NodesPanelCard
          panel={panel}
          readOnly={readOnly}
          onDelete={onDelete}
          onChange={onChange}
          onEditingChange={onEditingChange}
        />
      );
    case "table":
      return (
        <TablePanelCard
          panel={panel}
          readOnly={readOnly}
          onDelete={onDelete}
          onChange={onChange}
          onEditingChange={onEditingChange}
        />
      );
    case "chart":
      return (
        <ChartPanelCard
          panel={panel}
          readOnly={readOnly}
          onDelete={onDelete}
          onChange={onChange}
          onEditingChange={onEditingChange}
        />
      );
    case "number":
      return (
        <NumberPanelCard
          panel={panel}
          readOnly={readOnly}
          onDelete={onDelete}
          onChange={onChange}
          onEditingChange={onEditingChange}
        />
      );
    case "html":
      return (
        <HtmlPanelCard
          panel={panel}
          readOnly={readOnly}
          onDelete={onDelete}
          onChange={onChange}
          onEditingChange={onEditingChange}
        />
      );
    case "markdown":
    default:
      return (
        <MarkdownPanelCard
          panel={panel}
          readOnly={readOnly}
          onDelete={onDelete}
          onChange={onChange}
          onEditingChange={onEditingChange}
        />
      );
  }
}
