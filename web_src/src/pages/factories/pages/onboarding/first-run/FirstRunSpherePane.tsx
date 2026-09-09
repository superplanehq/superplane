import { cn } from "@/lib/utils";
import { useEffect, useRef } from "react";

import { FIRST_RUN_COPY } from "./firstRunCopy";

const CANVAS_SIZE = 300;

export type FirstRunSphereChip = { label: string; value: string; tone?: "ghost" | "amber" };

export type FirstRunSphereProps = {
  /** 0..1 — sphere density and brightness. */
  level: number;
  caption: string;
  /** Amber-highlighted prefix rendered before the caption, e.g. "Repository:". */
  captionHighlight?: string;
  leftChip?: FirstRunSphereChip;
  rightChip?: FirstRunSphereChip;
  phasesLit?: boolean;
  /** Redraw with flicker (analysis screen only). */
  animate?: boolean;
};

/**
 * Level drives a visible progression: a low level draws a small, sparse,
 * noisy cloud; a high level draws a large, dense sphere with concentric
 * rings, so the sphere gains structure with each completed step.
 */
function drawSphere(canvas: HTMLCanvasElement, level: number, tick: number) {
  const context = canvas.getContext("2d");
  if (!context || typeof context.clearRect !== "function") return;
  const size = canvas.width;
  const center = size / 2;
  const radius = (size / 2 - 8) * (0.6 + 0.4 * level);
  const cell = 7;
  context.clearRect(0, 0, size, size);
  for (let x = cell / 2; x < size; x += cell) {
    for (let y = cell / 2; y < size; y += cell) {
      const dx = x - center;
      const dy = y - center;
      const distance = Math.sqrt(dx * dx + dy * dy);
      if (distance > radius) continue;
      const seed = Math.sin(x * 12.99 + y * 78.23 + tick) * 43758.55;
      const random = seed - Math.floor(seed);
      const edge = distance / radius;
      const gapChance = 0.52 - 0.38 * level + edge * (0.5 - 0.22 * level) + (dy > 0 ? (dy / radius) * 0.25 : 0);
      if (random < gapChance) continue;
      const ring = 0.5 + 0.5 * Math.cos(edge * Math.PI * 7);
      const order = level * level;
      const structure = 1 - 0.55 * order * (1 - ring);
      const alpha = level * (0.5 + 0.5 * (1 - edge)) * (0.65 + 0.35 * random) * structure;
      context.fillStyle = `rgba(246, 168, 33, ${alpha.toFixed(3)})`;
      const dot = cell - 2.4;
      context.fillRect(x - dot / 2, y - dot / 2, dot, dot);
    }
  }
}

function SphereChip({ chip, side }: { chip: FirstRunSphereChip; side: "left" | "right" }) {
  return (
    <div
      className={cn(
        "absolute z-10 whitespace-nowrap rounded-lg border bg-background/90 px-3 py-2 font-mono text-[11px]",
        side === "left" ? "left-4 top-[24%]" : "right-4 top-[60%]",
        chip.tone === "ghost" && "border-dashed opacity-40",
        chip.tone === "amber" ? "border-[#f6a821]/70 text-[#f6a821]" : "border-border text-muted-foreground",
      )}
    >
      {chip.label}
      <span className={cn("mt-0.5 block text-[12px]", chip.tone === "amber" ? "text-[#f6a821]" : "text-foreground")}>
        {chip.value}
      </span>
    </div>
  );
}

export function FirstRunSpherePane({
  level,
  caption,
  captionHighlight,
  leftChip,
  rightChip,
  phasesLit = false,
  animate = false,
}: FirstRunSphereProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    drawSphere(canvas, level, 0);
    if (!animate) return;
    let tick = 0;
    const timer = window.setInterval(() => {
      tick += 0.7;
      drawSphere(canvas, level, tick);
    }, 260);
    return () => window.clearInterval(timer);
  }, [level, animate]);

  return (
    <div
      className="relative hidden overflow-hidden border-l border-border lg:flex lg:w-[44%]"
      style={{ background: "radial-gradient(ellipse at 50% 45%, #14100a 0%, #0d0c08 70%)" }}
    >
      <div className="absolute inset-0 flex items-center justify-center">
        {[340, 440, 540].map((diameter) => (
          <span
            key={diameter}
            className="absolute rounded-full border border-[#f6a821]/10"
            style={{ width: diameter, height: diameter, top: "47%", left: "50%", transform: "translate(-50%, -50%)" }}
          />
        ))}
        <canvas ref={canvasRef} width={CANVAS_SIZE} height={CANVAS_SIZE} className="relative -top-[3%]" />
      </div>
      {leftChip ? <SphereChip chip={leftChip} side="left" /> : null}
      {rightChip ? <SphereChip chip={rightChip} side="right" /> : null}
      <div className="absolute inset-x-0 bottom-11 z-10 flex justify-center gap-4 font-mono text-[10px] tracking-[0.08em]">
        {FIRST_RUN_COPY.sphere.phases.map((phase) => (
          <span
            key={phase}
            className={cn(
              "flex flex-col items-center gap-1.5 whitespace-nowrap",
              phasesLit ? "text-[#f6a821]" : "text-[#4d4a42]",
            )}
          >
            <i
              className={cn(
                "size-1.5 rounded-full",
                phasesLit ? "bg-[#f6a821] shadow-[0_0_8px_rgba(246,168,33,0.7)]" : "bg-[#3a382f]",
              )}
            />
            {phase}
          </span>
        ))}
      </div>
      <p className="absolute bottom-4 right-5 z-10 font-mono text-[11px] uppercase tracking-[0.14em] text-muted-foreground">
        {captionHighlight ? <span className="text-[#f6a821]">{captionHighlight} </span> : null}
        {caption}
      </p>
    </div>
  );
}
