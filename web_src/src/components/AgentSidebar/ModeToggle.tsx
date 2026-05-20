import { Compass, Hammer, Monitor } from "lucide-react";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { VISIBLE_AGENT_MODES, type AgentMode } from "./agentMode";

const modeConfig = {
  builder: {
    icon: Hammer,
    label: "Build",
    description: "Build mode — make changes to the canvas",
  },
  architect: {
    icon: Compass,
    label: "Plan",
    description: "Plan mode — design before building",
  },
  operator: {
    icon: Monitor,
    label: "Ask",
    description: "Ask mode — read-only questions and diagnostics",
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
  const activeLabel = modeConfig[mode].label;

  return (
    <div className="flex items-center gap-1.5" data-testid="agent-mode-toggle">
      <div className="flex items-center gap-0.5 rounded-md bg-slate-100 p-0.5">
        {VISIBLE_AGENT_MODES.map((key) => {
          const config = modeConfig[key];
          const Icon = config.icon;
          const isActive = mode === key;
          return (
            <Tooltip key={key}>
              <TooltipTrigger asChild>
                <button
                  type="button"
                  onClick={() => onSwitch(key)}
                  disabled={disabled || streaming}
                  className={cn(
                    "flex h-6 w-6 items-center justify-center rounded transition-colors",
                    isActive ? "bg-white text-slate-900 shadow-sm" : "text-slate-500 hover:text-slate-700",
                    (disabled || streaming) && !isActive && "cursor-not-allowed opacity-40",
                    isActive && streaming && "animate-pulse-border",
                  )}
                  aria-label={`${config.label} mode`}
                  aria-pressed={isActive}
                  data-testid={`agent-mode-${key}`}
                >
                  <Icon size={12} />
                </button>
              </TooltipTrigger>
              <TooltipContent side="top">{config.description}</TooltipContent>
            </Tooltip>
          );
        })}
      </div>
      <span className="text-xs font-medium text-slate-600">{activeLabel}</span>
    </div>
  );
}
