import type { ReactNode } from "react";

import { ConsoleContext, type ConsoleContextValue } from "../ConsoleContext";

import { mockConsoleContextValue } from "./storyFixtures";

/** Wrap children in a mock `ConsoleContext`. Pass overrides to vary behavior. */
export function MockConsoleProvider({
  children,
  value,
}: {
  children: ReactNode;
  value?: Partial<ConsoleContextValue>;
}) {
  return <ConsoleContext.Provider value={{ ...mockConsoleContextValue, ...value }}>{children}</ConsoleContext.Provider>;
}

/**
 * Fixed-size frame approximating a real dashboard grid cell so panels render at
 * realistic dimensions against the console's slate background.
 */
export function PanelFrame({
  children,
  width = 420,
  height = 280,
}: {
  children: ReactNode;
  width?: number;
  height?: number;
}) {
  return (
    <div className="bg-slate-100 p-4 dark:bg-gray-900">
      <div style={{ width, height }}>{children}</div>
    </div>
  );
}
