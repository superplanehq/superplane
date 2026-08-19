import { cn } from "@/lib/utils";
import { appDarkModeClasses } from "@/lib/appDarkModeClasses";
import { SEGMENTED_NAV_TAB_ACTIVE_CLASSES, SEGMENTED_NAV_TAB_INACTIVE_CLASSES } from "@/lib/segmentedNav";
import type { ThemePreference } from "@/lib/themePreference";
import { useTheme } from "@/contexts/useTheme";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Monitor, Moon, Sun } from "lucide-react";

const OPTIONS: Array<{ value: ThemePreference; label: string; Icon: typeof Sun }> = [
  { value: "light", label: "Light", Icon: Sun },
  { value: "dark", label: "Dark", Icon: Moon },
  { value: "system", label: "System", Icon: Monitor },
];

interface ThemePreferenceControlProps {
  /** `org` matches organization settings. `workspace` uses workspace sidebar tokens. */
  variant?: "org" | "workspace";
}

function toggleButtonClass(isActive: boolean, isWorkspace: boolean) {
  if (isWorkspace) {
    return isActive ? "bg-background text-foreground shadow-sm" : "text-muted-foreground hover:text-foreground";
  }
  return isActive ? SEGMENTED_NAV_TAB_ACTIVE_CLASSES : SEGMENTED_NAV_TAB_INACTIVE_CLASSES;
}

export function ThemePreferenceControl({ variant = "org" }: ThemePreferenceControlProps) {
  const { preference, setPreference } = useTheme();
  const isWorkspace = variant === "workspace";

  return (
    <div
      className={
        isWorkspace
          ? "mt-2 flex justify-center border-t border-sidebar-border pt-3"
          : cn("-mx-4 mt-2 border-t px-4 pt-4 pb-3", appDarkModeClasses.sidebarDivider)
      }
    >
      <div
        className={cn(
          "inline-flex h-8 w-fit gap-1 rounded-full p-1",
          isWorkspace ? "bg-muted" : "bg-slate-100 dark:bg-gray-800",
        )}
        role="group"
        aria-label="Appearance"
      >
        {OPTIONS.map(({ value, label, Icon }) => {
          const isActive = preference === value;

          return (
            <Tooltip key={value} delayDuration={350}>
              <TooltipTrigger asChild>
                <button
                  type="button"
                  aria-label={label}
                  aria-pressed={isActive}
                  data-testid={`theme-preference-${value}`}
                  onClick={() => setPreference(value)}
                  className={cn(
                    "flex size-6 items-center justify-center rounded-full transition-colors",
                    toggleButtonClass(isActive, isWorkspace),
                  )}
                >
                  <Icon className="h-3.5 w-3.5 shrink-0" aria-hidden />
                </button>
              </TooltipTrigger>
              <TooltipContent side="top">{label}</TooltipContent>
            </Tooltip>
          );
        })}
      </div>
    </div>
  );
}
