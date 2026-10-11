import { useEffect, useRef } from "react";

const CANVAS_SIZE = 320;
const CELL = 7;
const LOGO_PATH =
  "M543.228 153.905C626.85 108.604 728.51 169.153 728.511 264.256V392.792C728.511 399.008 728.04 405.156 727.144 411.195C805.606 385.932 890.542 443.979 890.542 530.912V659.447C890.541 705.5 865.32 747.86 824.824 769.798L536.299 926.102C452.678 971.4 351.017 910.853 351.016 815.75V687.215C351.016 680.996 351.482 674.845 352.378 668.802C273.918 694.061 188.986 636.026 188.984 549.095V420.56C188.985 374.504 214.213 332.146 254.707 310.209L543.228 153.905ZM402.646 671.62C401.575 676.69 401.016 681.911 401.016 687.215V815.75C401.017 872.963 462.174 909.387 512.48 882.137L654.951 804.95C647.141 803.568 639.333 800.898 631.777 796.771L402.646 671.62ZM840.542 530.912C840.542 473.696 779.384 437.272 729.077 464.525L593.306 538.07L677.759 584.198C718.042 606.201 743.101 648.444 743.101 694.344V730.731C743.101 741.47 740.933 751.471 737.09 760.453L801.006 725.834C825.365 712.637 840.541 687.155 840.542 659.447V530.912ZM429.683 629.418L655.742 752.889C672.57 762.08 693.1 758.895 693.101 739.723V694.344C693.101 666.73 678.027 641.317 653.794 628.08L543.423 567.796L429.683 629.418ZM278.525 354.173C254.163 367.371 238.985 392.854 238.984 420.56V549.095C238.986 606.308 300.142 642.729 350.449 615.477L489.658 540.057L416.729 500.545C376.234 478.608 351.006 436.25 351.006 390.194V353.71C351.007 334.846 357.679 318.249 368.423 305.467L278.525 354.173ZM438.286 331.507C421.462 322.394 401.007 325.778 401.006 344.912V390.194C401.006 417.9 416.185 443.382 440.547 456.581L539.653 510.267L653.955 448.348L438.286 331.507ZM678.511 264.256C678.51 207.042 617.352 170.618 567.046 197.87L417.485 278.89C432.1 277.13 447.558 279.662 462.104 287.543L677.617 404.295C678.2 400.523 678.511 396.68 678.511 392.792V264.256Z";

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
  const scale = (size * 0.72) / 810;
  context.translate((size - 720 * scale) / 2, (size - 810 * scale) / 2);
  context.scale(scale, scale);
  context.translate(-180, -135);
  context.fillStyle = "#000";
  context.fill(new Path2D(LOGO_PATH), "evenodd");
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
