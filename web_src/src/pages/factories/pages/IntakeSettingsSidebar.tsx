import { logoDarkInvertClass } from "@/lib/logoDarkMode";
import { cn } from "@/lib/utils";
import { Bot, Settings, Workflow, XIcon } from "lucide-react";

import { IntakeSettingsSaveAction, type IntakeSettingsSaveActionProps } from "./IntakeSourceSettingsFooter";
import { INTAKE_SETTINGS_COPY, intakeSettingsTabLabel, type IntakeSettingsTab } from "./intakeSourceSettingsModel";
import { lineIntakeSourceById, type LineIntakeSourceId } from "./lineIntakeModel";

const TAB_ICON = {
  general: Settings,
  agent: Bot,
  automation: Workflow,
} as const;

export function IntakeSettingsSidebar({
  sourceId,
  title,
  tabs,
  active,
  onSelect,
  onClose,
  save,
}: {
  sourceId: LineIntakeSourceId;
  title: string;
  tabs: readonly IntakeSettingsTab[];
  active: IntakeSettingsTab;
  onSelect: (tab: IntakeSettingsTab) => void;
  onClose: () => void;
  save?: IntakeSettingsSaveActionProps;
}) {
  const source = lineIntakeSourceById(sourceId);

  return (
    <header
      className="flex shrink-0 items-center justify-between gap-4 border-b border-sidebar-border bg-sidebar px-5 py-3 text-sidebar-foreground"
      data-testid="intake-settings-topbar"
    >
      <div className="flex min-w-0 items-center gap-4">
        <h2 className="flex min-w-0 shrink-0 items-center gap-2 text-[16px] font-semibold tracking-[-0.02em] text-sidebar-foreground">
          {source ? (
            <img
              src={source.iconSrc}
              alt=""
              aria-hidden
              data-testid="intake-settings-source-icon"
              className={cn(
                "size-4 shrink-0 object-contain",
                source.iconAlt === "GitHub" && "dark:brightness-0 dark:invert",
                logoDarkInvertClass(source.iconSrc),
              )}
            />
          ) : null}
          <span className="min-w-0 truncate">{title}</span>
        </h2>
        <nav aria-label={INTAKE_SETTINGS_COPY.tabsLabel} className="flex min-w-0 flex-wrap items-center gap-1">
          {tabs.map((tab) => {
            const Icon = TAB_ICON[tab];
            const selected = tab === active;
            return (
              <button
                key={tab}
                type="button"
                aria-current={selected ? "page" : undefined}
                className={cn(
                  "flex items-center gap-2 rounded-md px-2.5 py-1.5 text-[13px] font-medium",
                  selected
                    ? "bg-sidebar-accent text-sidebar-accent-foreground"
                    : "text-muted-foreground hover:bg-sidebar-accent hover:text-sidebar-accent-foreground",
                )}
                onClick={() => onSelect(tab)}
                data-testid={`intake-settings-tab-${tab}`}
              >
                <Icon className="size-4 shrink-0" aria-hidden />
                {intakeSettingsTabLabel(tab)}
              </button>
            );
          })}
        </nav>
      </div>
      <div className="flex shrink-0 items-center gap-3">
        {save ? <IntakeSettingsSaveAction {...save} /> : null}
        <button
          type="button"
          onClick={onClose}
          className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full hover:bg-slate-950/5 dark:hover:bg-white/10"
          aria-label="Close"
        >
          <XIcon className="h-4 w-4" aria-hidden />
        </button>
      </div>
    </header>
  );
}
