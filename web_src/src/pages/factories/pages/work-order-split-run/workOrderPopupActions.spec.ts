import { describe, expect, it } from "bun:test";

import { popupWorkOrderDisplayKey } from "./workOrderPopupActions";

describe("popupWorkOrderDisplayKey", () => {
  it("prefers the server-provided ticket key", () => {
    expect(popupWorkOrderDisplayKey({ key: "SP-42", number: "7" }, "RF")).toBe("SP-42");
  });

  it("composes the key from the workspace key and number", () => {
    expect(popupWorkOrderDisplayKey({ number: "42" }, "SP")).toBe("SP-42");
  });

  it("omits a placeholder when no identifier is available", () => {
    expect(popupWorkOrderDisplayKey({})).toBeUndefined();
  });
});
