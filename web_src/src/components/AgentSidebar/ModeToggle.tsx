import { Compass, Hammer, Monitor } from "lucide-react";
import { cn } from "@/lib/utils";
import type { AgentMode } from "./useAgentState";

const modeConfig = {
  builder: {
    icon: Hammer,
    label: "Build",
    activeText: "text-orange-700",
    activeBg: "bg-orange-50 border-orange-300",
    activeShadow: "shadow-sm shadow-orange-100",
  },
  architect: {
    icon: Compass,
    label: "Plan",
    activeText: "text-blue-700",
    activeBg: "bg-blue-50 border-blue-300",
    activeShadow: "shadow-sm shadow-blue-100",
  },
  operator: {
    icon: Monitor,
    label: "Ask",
    activeText: "text-emerald-700",
    activeBg: "bg-emerald-50 border-emerald-300",
    activeShadow: "shadow-sm shadow-emerald-100",
  },
} as const;

export function ModeToggle({
  mode,
  onSwitch,
  disabled,
  streaming,
}: {
  mode: AgentMode;
  onSwitch: (mode: AgentMode) => void;
  disabled?: boolean;
  streaming?: boolean;
}) {
  return (
    <div className="flex items-center bg-slate-100 rounded-md p-0.5 gap-0.5" data-testid="agent-mode-toggle">
      {(Object.keys(modeConfig) as AgentMode[]).map((key) => {
        const config = modeConfig[key];
        const Icon = config.icon;
        const isActive = mode === key;
        return (
          <button
            key={key}
            type="button"
            onClick={() => onSwitch(key)}
            disabled={disabled || streaming}
            className={cn(
              "flex items-center gap-1 px-2 py-1 rounded text-xs font-medium transition-all border border-transparent",
              isActive
                ? cn(config.activeText, config.activeBg, config.activeShadow)
                : "text-slate-500 hover:text-slate-700",
              (disabled || streaming) && !isActive && "opacity-40 cursor-not-allowed",
              isActive && streaming && "animate-pulse-border",
            )}
            aria-label={`${config.label} mode`}
            data-testid={`agent-mode-${key}`}
          >
            <Icon size={12} />
            {config.label}
          </button>
        );
      })}
    </div>
  );
}
