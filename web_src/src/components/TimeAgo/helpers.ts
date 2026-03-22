import React from "react";
import { TimeAgo } from "./TimeAgo";

/**
 * Creates a TimeAgo React element from a Date or string.
 * Use in .ts files where JSX is not available.
 */
export function renderTimeAgo(date: Date | string): React.ReactNode {
  return React.createElement(TimeAgo, { date });
}

/**
 * Creates a React element with a text prefix followed by a separator and a self-updating TimeAgo.
 * Use in .ts files where JSX is not available.
 */
export function renderWithTimeAgo(prefix: string, date: Date | string, separator = " · "): React.ReactNode {
  return React.createElement(React.Fragment, null, prefix, separator, React.createElement(TimeAgo, { date }));
}
