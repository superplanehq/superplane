/**
 * Picks an evenly spaced subset of day labels to render on a velocity
 * chart's x-axis, so labels never overlap no matter how narrow the chart
 * is.
 *
 * The label producers (`factoryVelocityFlow.ts` and
 * `describe_factory_velocity.go`) always emit a full label for every day.
 * This helper is the one place that decides how many of those labels
 * actually fit, given the chart's own measured width. That keeps the
 * thinning rule deterministic and unit-testable instead of depending on how
 * recharts happens to lay out a category axis.
 */

/** Pixel width a label like "Wed Jan 19" needs so it does not touch its neighbor. */
const MIN_LABEL_PX = 56;

/**
 * Returns the subset of `dayLabels` to render as ticks, always including the
 * first and last day.
 *
 * Returns label strings rather than indices so callers can pass the result
 * straight into recharts' `XAxis ticks` prop. This relies on every label in
 * `dayLabels` being unique within the window: the velocity page only ever
 * shows 14- or 30-day windows, and 30 consecutive calendar dates never repeat
 * a "weekday day" or "weekday month day" label, so the assumption holds. If a
 * producer ever emitted duplicate labels inside one window, matching ticks
 * back to their labels would collapse those duplicates into one tick.
 */
export function pickVelocityAxisTicks(dayLabels: string[], width: number): string[] {
  const safeWidth = Number.isFinite(width) && width > 0 ? width : 0;
  const maxTicks = Math.max(2, Math.floor(safeWidth / MIN_LABEL_PX));

  if (dayLabels.length <= maxTicks) {
    return dayLabels;
  }

  const lastIndex = dayLabels.length - 1;

  // An integer stride, rounded up, keeps every gap between chosen days at
  // least `stride` days apart. A fractional step rounded per-tick (e.g.
  // `Math.round(i * (lastIndex / (maxTicks - 1)))`) looks "more even" on
  // paper, but rounding drift can shrink one gap to a single day, which is
  // exactly the overlap this helper exists to prevent.
  const stride = Math.ceil(lastIndex / (maxTicks - 1));

  const indices: number[] = [];
  for (let index = 0; index <= lastIndex; index += stride) {
    indices.push(index);
  }

  if (indices[indices.length - 1] !== lastIndex) {
    const gapToLast = lastIndex - indices[indices.length - 1];
    // The final day never lines up exactly on the stride. Rather than tack
    // it on next to its neighbor (which could land closer than `stride`),
    // drop that neighbor when it would, so the last gap stays at least as
    // wide as every other one.
    if (gapToLast < stride && indices.length > 1) {
      indices.pop();
    }
    indices.push(lastIndex);
  }

  return indices.map((index) => dayLabels[index]);
}
