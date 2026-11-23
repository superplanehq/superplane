import { createContext } from "react";
import type { ColumnAutomation } from "../lib/columnAutomations";

export const AutomationRowContext = createContext<{
  move?: {
    start: (card: ColumnAutomation) => void;
    end: () => void;
    keyboard: (card: ColumnAutomation, direction: number) => void;
  };
  onRun?: (card: ColumnAutomation) => void;
  runPending?: boolean;
  runningAutomationId?: string;
}>({});
