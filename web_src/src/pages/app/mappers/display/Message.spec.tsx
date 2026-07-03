import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { Message } from "./Message";
import type { ExecutionInfo } from "../types";

function buildExecution(metadata: unknown): ExecutionInfo {
  return {
    id: "execution-1",
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
    resultReason: "RESULT_REASON_OK",
    resultMessage: "",
    metadata,
    configuration: {},
    rootEvent: undefined,
  };
}

describe("Message", () => {
  it("renders a fallback when metadata is undefined", () => {
    render(<Message lastExecution={buildExecution(undefined)} />);
    expect(screen.getByText("Empty message")).toBeInTheDocument();
  });

  it("renders a fallback when metadata is null", () => {
    render(<Message lastExecution={buildExecution(null)} />);
    expect(screen.getByText("Empty message")).toBeInTheDocument();
  });

  it("renders the provided message when metadata.message is a non-empty string", () => {
    render(<Message lastExecution={buildExecution({ message: "hello" })} />);
    expect(screen.getByText("hello")).toBeInTheDocument();
  });
});
