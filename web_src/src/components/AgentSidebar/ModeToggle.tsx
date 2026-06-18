import { useCallback, useEffect, useLayoutEffect, useRef, useState } from "react";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { VISIBLE_AGENT_MODES, type AgentMode } from "./agentMode";

const modeConfig = {
  builder: {
    label: "Build",
    description: "Build mode — make changes to the canvas",
  },
  operator: {
    label: "Ask",
    description: "Ask mode — read-only questions and diagnostics",
  },
} as const;

function indicatorClasses(mode: AgentMode): string {
  if (mode === "builder") return "border-0 bg-[var(--purple)]";
  return "bg-slate-500 border-transparent";
}

function labelColor(key: AgentMode, isActive: boolean): string {
  if (!isActive) return "text-slate-600 hover:text-slate-700";
  if (key === "builder") return "text-white";
  return "text-white";
}

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
  const containerRef = useRef<HTMLDivElement>(null);
  const buttonRefs = useRef<Partial<Record<AgentMode, HTMLButtonElement>>>({});
  const [indicator, setIndicator] = useState({ width: 0, left: 0 });

  const updateIndicator = useCallback(() => {
    const container = containerRef.current;
    const button = buttonRefs.current[mode];
    if (!container || !button) return;

    const containerRect = container.getBoundingClientRect();
    const buttonRect = button.getBoundingClientRect();
    setIndicator({
      width: buttonRect.width,
      left: buttonRect.left - containerRect.left,
    });
  }, [mode]);

  useLayoutEffect(() => {
    updateIndicator();
  }, [updateIndicator]);

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    const observer = new ResizeObserver(() => updateIndicator());
    observer.observe(container);
    return () => observer.disconnect();
  }, [updateIndicator]);

  const handleSwitch = useCallback(
    (next: AgentMode) => {
      if (disabled || streaming || next === mode) return;
      onSwitch(next);
    },
    [disabled, streaming, mode, onSwitch],
  );

  return (
    <div
      ref={containerRef}
      className="relative inline-flex items-center rounded-full bg-slate-200"
      data-testid="agent-mode-toggle"
    >
      <div
        aria-hidden
        className={cn(
          "pointer-events-none absolute inset-y-0 left-0 rounded-full border will-change-[transform,width,background-color,border-color]",
          "transition-[transform,width,background-color,border-color] duration-200 ease-out",
          indicatorClasses(mode),
        )}
        style={{
          width: indicator.width,
          transform: `translateX(${indicator.left}px)`,
        }}
      />

      {VISIBLE_AGENT_MODES.map((key) => {
        const config = modeConfig[key];
        const isActive = mode === key;
        return (
          <Tooltip key={key}>
            <TooltipTrigger asChild>
              <button
                ref={(node) => {
                  if (node) buttonRefs.current[key] = node;
                }}
                type="button"
                onClick={() => handleSwitch(key)}
                disabled={disabled || streaming}
                className={cn(
                  "relative z-10 rounded-full px-2 py-1 text-xs font-medium leading-none transition-colors duration-200 ease-out",
                  labelColor(key, isActive),
                  (disabled || streaming) && !isActive && "cursor-not-allowed opacity-40",
                )}
                aria-label={`${config.label} mode`}
                aria-pressed={isActive}
                data-testid={`agent-mode-${key}`}
              >
                {config.label}
              </button>
            </TooltipTrigger>
            <TooltipContent side="top">{config.description}</TooltipContent>
          </Tooltip>
        );
      })}
    </div>
  );
}
