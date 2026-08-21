/** Shared factory sidebar surface, font, and control chrome. */
export const factorySidebarFontClassName =
  "factory-sidebar font-inter [font-feature-settings:'cv02','cv03','cv04','cv11'] text-[13px] font-normal leading-[19.5px] tracking-[-0.01em] antialiased";

export const factorySidebarSurfaceClassName =
  "border-border bg-background text-foreground dark:border-border dark:bg-background dark:text-foreground";

export const factorySidebarHeadingClassName = "min-w-0 text-[13px] font-semibold tracking-[-0.01em] text-foreground";

export const factorySidebarCloseButtonClassName =
  "flex h-7 w-7 shrink-0 items-center justify-center rounded-full text-muted-foreground transition-colors hover:bg-accent hover:text-foreground";

export const factorySidebarResizeLineClassName =
  "pointer-events-none absolute top-0 bottom-0 left-1/2 w-px -translate-x-1/2 bg-transparent transition-colors group-hover:bg-foreground/40";

export const factorySidebarMutedIconClassName = "text-muted-foreground";

export const factorySidebarInputClassName =
  "h-8 border-border bg-background pl-9 text-[13px] text-foreground shadow-none placeholder:text-muted-foreground focus:border-ring dark:border-border dark:bg-background dark:text-foreground dark:placeholder:text-muted-foreground dark:focus:border-ring";

export const factorySidebarIconButtonClassName =
  "flex size-8 shrink-0 items-center justify-center rounded-md border border-border bg-background text-muted-foreground transition-colors hover:bg-accent hover:text-foreground dark:border-border dark:bg-background dark:text-muted-foreground dark:hover:bg-accent dark:hover:text-foreground";

export const factorySidebarRowHoverClassName = "hover:bg-accent/70 dark:hover:bg-accent/70";

export function factorySidebarKindLabelClassName(kind: string): string {
  if (kind === "trigger") {
    return "text-[11px] font-medium text-[#2563eb]";
  }
  return "text-[11px] font-medium text-[#16a34a]";
}
