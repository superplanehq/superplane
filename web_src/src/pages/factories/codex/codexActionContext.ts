import { createContext, useContext } from "react";
import { getApiErrorReason } from "@/lib/errors";

export class CodexActionCancelled extends Error {
  constructor() {
    super("Codex connection canceled.");
    this.name = "CodexActionCancelled";
  }
}

export function isCodexActionCancelled(error: unknown): boolean {
  return error instanceof CodexActionCancelled;
}

export const CodexActionContext = createContext<
  ((organizationId: string, factoryId: string) => Promise<boolean>) | null
>(null);

/** Let the server validate ownership restrictions before offering a connection. */
export function useCodexActionGuard(organizationId: string, factoryId: string) {
  const connect = useContext(CodexActionContext);
  return async <Result>(action: () => Promise<Result>): Promise<Result> => {
    try {
      return await action();
    } catch (error) {
      if (!connect || getApiErrorReason(error) !== "CODEX_CONNECTION_REQUIRED") throw error;
      if (!(await connect(organizationId, factoryId))) throw new CodexActionCancelled();
      // The request runs all validation again against the refreshed connection.
      return action();
    }
  };
}
