import { afterEach, beforeEach, describe, expect, it } from "bun:test";

import {
  needsStartConfirm,
  persistSkipStartConfirm,
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
    expect(startConfirmTone(undefined)).toBe("missing");
    expect(startConfirmBody(undefined)).toBe(START_CONFIRM_COPY.missing);
    expect(needsStartConfirm(undefined)).toBe(true);
  });

  it("warns when the score is below 3", () => {
    expect(startConfirmTone(0)).toBe("low");
    expect(startConfirmTone(2)).toBe("low");
    expect(startConfirmBody(2)).toBe(START_CONFIRM_COPY.low);
    expect(needsStartConfirm(2)).toBe(true);
  });

  it("cautions when the score is 3 or 4", () => {
    expect(startConfirmTone(3)).toBe("mid");
    expect(startConfirmTone(4)).toBe("mid");
    expect(startConfirmBody(4)).toBe(START_CONFIRM_COPY.mid);
    expect(needsStartConfirm(4)).toBe(true);
  });

  it("skips the dialog at score 5", () => {
    expect(startConfirmTone(5)).toBeUndefined();
    expect(startConfirmBody(5)).toBeUndefined();
    expect(needsStartConfirm(5)).toBe(false);
  });

  it("honors Do not ask again from localStorage", () => {
    persistSkipStartConfirm();
    expect(window.localStorage.getItem(START_CONFIRM_STORAGE_KEY)).toBe("1");
    expect(needsStartConfirm(undefined)).toBe(false);
    expect(needsStartConfirm(1)).toBe(false);
    expect(needsStartConfirm(4)).toBe(false);
  });
});
