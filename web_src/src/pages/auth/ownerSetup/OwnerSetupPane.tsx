import { useEffect, useRef } from "react";

const CANVAS_SIZE = 320;
const CELL = 7;
const LOGO_PATH =
  "M29 14.32C28.5655 14.7542 28.1107 15.1686 27.6379 15.5626L17.112 5.16914L22.7409 18.5877C22.1782 18.8335 21.6021 19.0552 21.0139 19.2508L15.4286 5.93611V20.2791C15.1216 20.2926 14.8129 20.3 14.5025 20.3L14.2391 20.2982C14.0172 20.2955 13.7961 20.2888 13.5759 20.2791V5.88893L7.97322 19.2446C7.38516 19.0484 6.80914 18.8264 6.24652 18.58L11.8656 5.18536L1.36087 15.5578C0.888333 15.1638 0.434276 14.7489 0 14.3147L14.4975 0L29 14.32Z";

const RINGS = [260, 480];

function ownerSetupMarkLevel(step: number, stepCount: number) {
  if (stepCount < 1) return 1;
  return Math.min(1, Math.max(0, step / stepCount));
}

function logoMask(size: number) {
  const mask = document.createElement("canvas");
  mask.width = size;
  mask.height = size;
  const context = mask.getContext("2d");
  if (!context || typeof Path2D === "undefined") return null;
  const scale = (size * 0.62) / 29;
  context.translate((size - 29 * scale) / 2, (size - 21 * scale) / 2);
  context.scale(scale, scale);
  context.fillStyle = "#000";
  context.fill(new Path2D(LOGO_PATH));
  return erodeMask(context.getImageData(0, 0, size, size).data, size, 2);
}

function erodeMask(source: Uint8ClampedArray, size: number, radius: number) {
  const eroded = new Uint8ClampedArray(source.length);
  for (let y = radius; y < size - radius; y += 1) {
    for (let x = radius; x < size - radius; x += 1) {
      if (neighborhoodFilled(source, size, x, y, radius)) eroded[(y * size + x) * 4 + 3] = 255;
    }
  }
  return eroded;
}

function neighborhoodFilled(source: Uint8ClampedArray, size: number, x: number, y: number, radius: number) {
  for (let dy = -radius; dy <= radius; dy += 1) {
    for (let dx = -radius; dx <= radius; dx += 1) {
      if (source[((y + dy) * size + (x + dx)) * 4 + 3] < 40) return false;
    }
  }
  return true;
}

function cellHitsLogo(mask: Uint8ClampedArray, size: number, x: number, y: number) {
  const left = Math.max(0, Math.floor(x - CELL / 2));
  const top = Math.max(0, Math.floor(y - CELL / 2));
  const right = Math.min(size - 1, Math.ceil(x + CELL / 2));
  const bottom = Math.min(size - 1, Math.ceil(y + CELL / 2));
  for (let py = top; py <= bottom; py += 1) {
    for (let px = left; px <= right; px += 1) {
      if (mask[(py * size + px) * 4 + 3] >= 40) return true;
    }
  }
  return false;
}

function drawPixelMark(canvas: HTMLCanvasElement, mask: Uint8ClampedArray | null, level: number) {
  const context = canvas.getContext("2d");
  if (!context || typeof context.clearRect !== "function" || !mask) return;
  const dark = canvas.closest(".dark") !== null;
  const size = canvas.width;
  const alpha = 0.22 + 0.78 * level * level;
  context.clearRect(0, 0, size, size);
  context.fillStyle = dark ? `rgba(246, 168, 33, ${alpha.toFixed(3)})` : `rgba(91, 51, 173, ${alpha.toFixed(3)})`;
  const dot = CELL - 2.4;
  for (let x = CELL / 2; x < size; x += CELL) {
    for (let y = CELL / 2; y < size; y += CELL) {
      if (!cellHitsLogo(mask, size, x, y)) continue;
      context.fillRect(x - dot / 2, y - dot / 2, dot, dot);
    }
  }
}

export function OwnerSetupPane({ caption, step, stepCount }: { caption: string; step: number; stepCount: number }) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const level = ownerSetupMarkLevel(step, stepCount);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const mask = logoMask(canvas.width);
    const draw = () => drawPixelMark(canvas, mask, level);
    draw();
    const observer = new MutationObserver(draw);
    observer.observe(document.documentElement, { attributes: true, attributeFilter: ["class"] });
    return () => observer.disconnect();
  }, [level]);

  return (
    <div
      data-testid="owner-setup-pane"
      data-step={step}
      data-step-count={stepCount}
      className="relative hidden shrink-0 overflow-hidden border-l border-border bg-[radial-gradient(ellipse_at_50%_45%,#faf9fd_0%,#f1eff7_70%)] lg:flex lg:w-[44%] lg:max-w-[720px] dark:bg-[radial-gradient(ellipse_at_50%_45%,#14100a_0%,#0d0c08_70%)]"
    >
      <div className="absolute inset-0 flex items-center justify-center">
        {RINGS.map((diameter) => (
          <span
            key={diameter}
            className="absolute rounded-full border border-[#5b33ad]/15 dark:border-[#f6a821]/10"
            style={{ width: diameter, height: diameter, opacity: 0.16 + 0.84 * level * level }}
          />
        ))}
        <canvas ref={canvasRef} width={CANVAS_SIZE} height={CANVAS_SIZE} className="relative" />
      </div>
      <p className="absolute bottom-4 right-5 font-mono text-[11px] uppercase tracking-[0.14em] text-muted-foreground">
        {caption}
      </p>
    </div>
  );
}
