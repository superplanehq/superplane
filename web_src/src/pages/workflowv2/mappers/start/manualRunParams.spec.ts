import { describe, expect, it } from "vitest";

import { extractManualRunParams, hasManualRunParams, mergeManualRunPayload, parseParamString } from "./manualRunParams";

describe("parseParamString", () => {
  it("parses string and select params from issue examples", () => {
    const name = parseParamString(
      "param(type:string, title:'Enter a machine name', default:'machine-1', required:false)",
    );
    expect("error" in name).toBe(false);
    if ("error" in name) return;
    expect(name.isParam).toBe(true);
    expect(name.def.type).toBe("string");
    expect(name.def.title).toBe("Enter a machine name");
    expect(name.def.default).toBe("machine-1");
    expect(name.def.required).toBe(false);

    const size = parseParamString(
      "param(type:select, values:'2 vCPU|4 vCPU|8 vCPU', title:'Select size', required:true)",
    );
    expect("error" in size).toBe(false);
    if ("error" in size) return;
    expect(size.def.type).toBe("select");
    expect(size.def.values).toEqual(["2 vCPU", "4 vCPU", "8 vCPU"]);
    expect(size.def.required).toBe(true);
  });

  it("returns isParam false for static values", () => {
    const result = parseParamString("machine-1");
    expect("error" in result).toBe(false);
    if ("error" in result) return;
    expect(result.isParam).toBe(false);
  });
});

describe("hasManualRunParams / mergeManualRunPayload", () => {
  const staticPayload = {
    body: { name: "machine-1", size: "2 vCPU" },
  };

  it("detects no params in all-static template", () => {
    expect(hasManualRunParams(staticPayload)).toBe(false);
    expect(extractManualRunParams(staticPayload)).toEqual([]);
  });

  it("merges mixed static and parameterized fields", () => {
    const mixed = {
      body: {
        name: "param(type:string, title:'Enter a machine name', default:'machine-1', required:false)",
        size: "2 vCPU",
      },
    };
    expect(hasManualRunParams(mixed)).toBe(true);
    const fields = extractManualRunParams(mixed);
    expect(fields).toHaveLength(1);
    expect(fields[0].path).toBe("body.name");

    const merged = mergeManualRunPayload(mixed, { "body.name": "machine-9" });
    expect(merged.error).toBeUndefined();
    expect(merged.payload?.body).toEqual({ name: "machine-9", size: "2 vCPU" });
  });

  it("merges multiple parameters", () => {
    const multi = {
      body: {
        name: "param(type:string, title:'Enter a machine name', default:'machine-1', required:false)",
        size: "param(type:select, values:'2 vCPU|4 vCPU|8 vCPU', title:'Select size', required:true)",
      },
    };
    const merged = mergeManualRunPayload(multi, {
      "body.name": "machine-2",
      "body.size": "4 vCPU",
    });
    expect(merged.payload?.body).toEqual({ name: "machine-2", size: "4 vCPU" });
  });

  it("uses defaults when values omitted", () => {
    const tmpl = {
      name: "param(type:string, title:'Name', default:'machine-1', required:false)",
    };
    const merged = mergeManualRunPayload(tmpl, {});
    expect(merged.payload).toEqual({ name: "machine-1" });
  });

  it("errors on missing required parameter", () => {
    const tmpl = {
      size: "param(type:select, values:'2 vCPU|4 vCPU', title:'Size', required:true)",
    };
    const merged = mergeManualRunPayload(tmpl, {});
    expect(merged.error).toContain("size");
  });
});
