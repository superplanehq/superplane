import { afterEach, beforeEach, describe, expect, it } from "bun:test";

import {
  needsStartConfirm,
  persistSkipStartConfirm,
  refineAgentIsWorking,
  START_CONFIRM_COPY,
  START_CONFIRM_STORAGE_KEY,
  startConfirmBody,
  startConfirmTone,
} from "./startConfirm";

describe("startConfirm", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  afterEach(() => {
    window.localStorage.clear();
  });

  it("asks before a score exists", () => {
    expect(startConfirmTone({})).toBe("missing");
    expect(startConfirmBody({})).toBe(START_CONFIRM_COPY.missing);
    expect(needsStartConfirm({})).toBe(true);
  });

  it("warns when Clarity is 2 or lower", () => {
    expect(startConfirmTone({ clarity: 0, confidence: 5 })).toBe("low");
    expect(startConfirmTone({ clarity: 2, confidence: 5 })).toBe("low");
    expect(startConfirmBody({ clarity: 2 })).toBe(START_CONFIRM_COPY.low);
    expect(needsStartConfirm({ clarity: 2 })).toBe(true);
  });

  it("cautions when Confidence is low or either score is 3", () => {
    expect(startConfirmTone({ clarity: 5, confidence: 2 })).toBe("mid");
    expect(startConfirmTone({ clarity: 3, confidence: 5 })).toBe("mid");
    expect(startConfirmTone({ clarity: 5, confidence: 3 })).toBe("mid");
    expect(startConfirmBody({ clarity: 5, confidence: 3 })).toBe(START_CONFIRM_COPY.mid);
    expect(needsStartConfirm({ clarity: 5, confidence: 3 })).toBe(true);
  });

  it("skips the dialog when every known score is 4 or 5", () => {
    expect(startConfirmTone({ clarity: 4, confidence: 5 })).toBeUndefined();
    expect(startConfirmBody({ clarity: 5, confidence: 4 })).toBeUndefined();
    expect(needsStartConfirm({ confidence: 5 })).toBe(false);
  });

  it("warns while the agent still works, even when scores are high", () => {
    expect(startConfirmTone({ clarity: 4, confidence: 5, agentWorking: true })).toBe("working");
    expect(startConfirmBody({ clarity: 4, confidence: 5, agentWorking: true })).toBe(START_CONFIRM_COPY.working);
    expect(needsStartConfirm({ clarity: 4, confidence: 5, agentWorking: true })).toBe(true);
  });

  it("honors Do not ask again from localStorage", () => {
    persistSkipStartConfirm();
    expect(window.localStorage.getItem(START_CONFIRM_STORAGE_KEY)).toBe("1");
    expect(needsStartConfirm({})).toBe(false);
    expect(needsStartConfirm({ clarity: 1 })).toBe(false);
    expect(needsStartConfirm({ clarity: 3, confidence: 3 })).toBe(false);
  });

  it("does not let Do not ask again skip the still-working confirm", () => {
    persistSkipStartConfirm();
    expect(needsStartConfirm({ clarity: 4, confidence: 5, agentWorking: true })).toBe(true);
  });

  it("does not treat the empty session fallback as a working agent", () => {
    expect(refineAgentIsWorking({ hasSession: false, machineStatus: "starting" })).toBe(false);
    expect(refineAgentIsWorking({ hasSession: false, machineStatus: "running" })).toBe(false);
    expect(needsStartConfirm({ clarity: 4, confidence: 5, agentWorking: false })).toBe(false);
  });

  it("treats starting and running as working only when a session exists", () => {
    expect(refineAgentIsWorking({ hasSession: true, machineStatus: "starting" })).toBe(true);
    expect(refineAgentIsWorking({ hasSession: true, machineStatus: "running" })).toBe(true);
    expect(refineAgentIsWorking({ hasSession: true, machineStatus: "waiting" })).toBe(false);
  });
});
