import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import type { ExecutionInfo } from "../types";
import { Message } from "./Message";

function makeExecution(overrides?: Partial<ExecutionInfo>): ExecutionInfo {
  const now = new Date().toISOString();

  return {
    id: "execution-1",
    createdAt: now,
    updatedAt: now,
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
    resultReason: "RESULT_REASON_OK",
    resultMessage: "",
    metadata: { message: "Hello world", color: "green" },
    configuration: {},
    rootEvent: {
      id: "event-1",
      createdAt: now,
      customName: "Start event",
      data: {},
      nodeId: "trigger-1",
      type: "trigger",
    },
    ...overrides,
  };
}

describe("display Message", () => {
  it("renders null when lastExecution is null", () => {
    const { container } = render(<Message lastExecution={null} />);
    expect(container).toBeEmptyDOMElement();
  });

  it("renders the metadata message and color when present", () => {
    render(<Message lastExecution={makeExecution()} />);
    expect(screen.getByText("Hello world")).toBeInTheDocument();
  });

  it("falls back to defaults when metadata is undefined", () => {
    const execution = makeExecution({ metadata: undefined });
    expect(() => render(<Message lastExecution={execution} />)).not.toThrow();
    expect(screen.getByText("Empty message")).toBeInTheDocument();
  });

  it("falls back to defaults when metadata is null", () => {
    const execution = makeExecution({ metadata: null });
    expect(() => render(<Message lastExecution={execution} />)).not.toThrow();
    expect(screen.getByText("Empty message")).toBeInTheDocument();
  });

  it("falls back to defaults when metadata is missing keys", () => {
    const execution = makeExecution({ metadata: {} });
    render(<Message lastExecution={execution} />);
    expect(screen.getByText("Empty message")).toBeInTheDocument();
  });
});
