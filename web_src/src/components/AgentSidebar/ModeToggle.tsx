import { Compass, Hammer, Monitor } from "lucide-react";
import { cn } from "@/lib/utils";
import type { AgentMode } from "./useAgentState";

export function ModeToggle({
  mode,
  onSwitch,
  disabled,
}: {
  mode: AgentMode;
  onSwitch: (mode: AgentMode) => void;
  disabled?: boolean;
}) {
  return (
    <div
      className={cn(
        "flex items-center bg-slate-100 rounded-md p-0.5 gap-0.5",
        disabled && "opacity-50 pointer-events-none",
      )}
      data-testid="agent-mode-toggle"
    >
      <button
        type="button"
        onClick={() => onSwitch("builder")}
        disabled={disabled}
        className={cn(
          "flex items-center gap-1 px-2 py-1 rounded text-xs font-medium transition-colors",
          mode === "builder" ? "bg-white text-violet-700 shadow-sm" : "text-slate-500 hover:text-slate-700",
        )}
        aria-label="Builder mode"
        data-testid="agent-mode-builder"
      >
        <Hammer size={12} />
        Builder
      </button>
      <button
        type="button"
        onClick={() => onSwitch("architect")}
        disabled={disabled}
        className={cn(
          "flex items-center gap-1 px-2 py-1 rounded text-xs font-medium transition-colors",
          mode === "architect" ? "bg-white text-violet-700 shadow-sm" : "text-slate-500 hover:text-slate-700",
        )}
        aria-label="Architect mode"
        data-testid="agent-mode-architect"
      >
        <Compass size={12} />
        Architect
      </button>
      <button
        type="button"
        onClick={() => onSwitch("operator")}
        disabled={disabled}
        className={cn(
          "flex items-center gap-1 px-2 py-1 rounded text-xs font-medium transition-colors",
          mode === "operator" ? "bg-white text-violet-700 shadow-sm" : "text-slate-500 hover:text-slate-700",
        )}
        aria-label="Operator mode"
        data-testid="agent-mode-operator"
      >
        <Monitor size={12} />
        Operator
      </button>
    </div>
  );
}
