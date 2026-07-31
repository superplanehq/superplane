import { cn } from "@/lib/utils";

export const createActionCardClassName = cn(
  "relative flex w-full flex-row items-center gap-4 rounded-md border border-dashed border-status-success-edge bg-status-success-subtle px-4 py-3 transition-colors",
  "hover:bg-status-success-edge",
);

export const createActionCardDisabledClassName = cn(
  "relative flex w-full flex-row items-center gap-4 rounded-md border border-dashed border-edge-strong bg-surface-subtle px-4 py-3 text-content-muted transition-colors cursor-not-allowed",
);

export const createActionIconClassName =
  "flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-status-success text-content-inverse";

export const createActionIconDisabledClassName =
  "flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-action-neutral-hover text-content-primary";
