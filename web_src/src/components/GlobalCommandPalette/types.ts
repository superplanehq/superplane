import type { CanvasesCanvasSummary } from "@/api-client";
import type { LucideIcon } from "lucide-react";

export type CommandPage = "root" | "organization-settings" | "canvas-settings" | "open-canvas" | "admin";

export type PermissionCheck = {
  resource: string;
  action: string;
};

export type PaletteAction = {
  id: string;
  label: string;
  description?: string;
  icon: LucideIcon;
  keywords?: string[];
  shortcut?: string;
  disabled?: boolean;
  onSelect: () => void;
};

export type PalettePageAction = Omit<PaletteAction, "onSelect"> & {
  page: CommandPage;
};

export type CanvasCommandListProps = {
  canvases: CanvasesCanvasSummary[];
  canvasesLoading: boolean;
  organizationId: string | null;
  goTo: (href: string) => void;
};
