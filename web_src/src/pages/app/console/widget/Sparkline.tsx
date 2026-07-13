/**
 * Compact SVG sparkline shared by number and scorecard panels.
 *
 * Renders a filled area path with the same coordinate math the pre-shared
 * copy used, plus a `className` hook so callers can drive the stroke /
 * fill color via `currentColor` (the number panel uses the default sky
 * tint; the scorecard passes emerald / red / muted based on its status).
 */

export interface SparklineProps {
  values: number[];
  width?: number;
  height?: number;
  /**
   * Tailwind `text-*` class used to color the stroke and area fill via
   * `currentColor`. Defaults to the sky/indigo tint historically used by
   * the number widget.
   */
  className?: string;
}

const DEFAULT_WIDTH = 120;
const DEFAULT_HEIGHT = 28;
const STROKE_WIDTH = 1.5;
const DEFAULT_CLASS = "text-sky-500 dark:text-indigo-400";

export function Sparkline({ values, width = DEFAULT_WIDTH, height = DEFAULT_HEIGHT, className }: SparklineProps) {
  // Inset the plot so round joins / stroke width aren't clipped at the SVG edges.
  const padY = Math.ceil(STROKE_WIDTH / 2) + 1;
  const plotTop = padY;
  const plotBottom = height - padY;
  const plotHeight = plotBottom - plotTop;
  const min = Math.min(...values);
  const max = Math.max(...values);
  const range = max - min || 1;
  const stepX = values.length > 1 ? width / (values.length - 1) : 0;
  const linePoints = values.map((v, i) => {
    const x = i * stepX;
    const y = plotTop + plotHeight - ((v - min) / range) * plotHeight;
    return `${x.toFixed(1)},${y.toFixed(1)}`;
  });
  const lineCoords = linePoints.join(" ");
  const firstX = (0).toFixed(1);
  const lastX = ((values.length - 1) * stepX).toFixed(1);
  const baselineY = plotBottom.toFixed(1);
  const areaPath = `M${linePoints[0]} L${linePoints.slice(1).join(" L")} L${lastX},${baselineY} L${firstX},${baselineY} Z`;
  return (
    <svg
      width={width}
      height={height}
      className={`block ${className ?? DEFAULT_CLASS}`}
      viewBox={`0 0 ${width} ${height}`}
      aria-hidden
    >
      <path d={areaPath} fill="currentColor" fillOpacity={0.2} stroke="none" />
      <polyline
        points={lineCoords}
        fill="none"
        stroke="currentColor"
        strokeWidth={STROKE_WIDTH}
        strokeLinejoin="round"
        strokeLinecap="round"
      />
    </svg>
  );
}
