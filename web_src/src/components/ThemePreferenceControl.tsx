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

export function ThemePreferenceControl() {
  const { preference, setPreference } = useTheme();

  return (
    <div className={cn("-mx-4 mt-2 border-t px-4 pt-4 pb-3", appDarkModeClasses.sidebarDivider)}>
      <div className="inline-flex h-8 w-fit gap-1 rounded-full bg-slate-100 p-1 dark:bg-gray-800">
        {OPTIONS.map(({ value, label, Icon }) => {
          const isActive = preference === value;

          return (
            <Tooltip key={value} delayDuration={350}>
              <TooltipTrigger asChild>
                <button
                  type="button"
                  aria-label={label}
                  aria-pressed={isActive}
                  onClick={() => setPreference(value)}
                  className={cn(
                    "flex size-6 items-center justify-center rounded-full transition-colors",
                    isActive ? SEGMENTED_NAV_TAB_ACTIVE_CLASSES : SEGMENTED_NAV_TAB_INACTIVE_CLASSES,
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
