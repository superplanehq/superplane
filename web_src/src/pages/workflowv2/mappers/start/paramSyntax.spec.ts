import { describe, expect, it } from "vitest";
import { hasParams, issueExamplePayload, isParamString, parseParamString, parseParams } from "./paramSyntax";

describe("paramSyntax", () => {
  it("detects param leaves", () => {
    expect(hasParams(issueExamplePayload())).toBe(true);
    expect(hasParams({ message: "hello" })).toBe(false);
  });

  it("parses issue example payload", () => {
    const defs = parseParams(issueExamplePayload());
    expect(defs).toHaveLength(2);
    expect(defs.map((def) => def.path)).toEqual(["body.name", "body.size"]);
    expect(defs[0]?.order).toBe(1);
    expect(defs[1]?.order).toBe(2);
  });

  it("recognizes param expressions", () => {
    expect(isParamString("  param(type:string)  ")).toBe(true);
    expect(isParamString("param type string")).toBe(false);
    expect(isParamString("static value")).toBe(false);
  });

  it("parses issue example param strings", () => {
    const name = parseParamString(
      "body.name",
      "param(type:string, title:'Enter a machine name', default:'machine-1', required:false, order:1)",
    );
    expect(name.path).toBe("body.name");
    expect(name.type).toBe("string");
    expect(name.title).toBe("Enter a machine name");
    expect(name.default).toBe("machine-1");
    expect(name.required).toBe(false);
    expect(name.order).toBe(1);

    const size = parseParamString(
      "body.size",
      "param(type:select, values:'2 vCPU|4 vCPU|8 vCPU', title:'Select size', required:true, order:2)",
    );
    expect(size.path).toBe("body.size");
    expect(size.type).toBe("select");
    expect(size.title).toBe("Select size");
    expect(size.required).toBe(true);
    expect(size.values).toEqual(["2 vCPU", "4 vCPU", "8 vCPU"]);
    expect(size.order).toBe(2);
  });

  it("defaults omitted order to zero", () => {
    const def = parseParamString("body.name", "param(type:string, required:false)");
    expect(def.order).toBe(0);
  });

  it("sorts definitions by order then path", () => {
    const defs = parseParams({
      z: "param(type:string, order:2)",
      a: "param(type:string, order:1)",
      m: "param(type:string, order:1)",
    });
    expect(defs.map((def) => def.path)).toEqual(["a", "m", "z"]);
  });

  it("rejects invalid order tokens", () => {
    expect(() => parseParamString("x", "param(type:string, order:-1)")).toThrow(/order:/);
    expect(() => parseParamString("x", "param(type:string, order:1.5)")).toThrow(/order:/);
    expect(() => parseParamString("x", "param(type:string, order:abc)")).toThrow(/order:/);
  });

  it("rejects invalid quoted charset", () => {
    expect(() => parseParamString("body.name", "param(type:string, title:'bad,comma')")).toThrow(/comma/);
    expect(() => parseParamString("body.name", "param(type:string, title:'bad\"quote')")).toThrow();
    expect(() => parseParamString("body.name", "param(type:string, title:'bad\\'quote')")).toThrow();
  });

  it("rejects malformed expressions", () => {
    expect(() => parseParamString("body.name", "not a param")).toThrow();
    expect(() => parseParamString("body.name", "param()")).toThrow();
    expect(() => parseParamString("body.name", "param(type:string, title:'unterminated")).toThrow();
  });

  it("parses boolean and number params", () => {
    const enabled = parseParamString("enabled", "param(type:boolean, required:true, default:false)");
    expect(enabled.type).toBe("boolean");
    expect(enabled.default).toBe(false);
    expect(enabled.required).toBe(true);

    const count = parseParamString("count", "param(type:number, default:42)");
    expect(count.type).toBe("number");
    expect(count.default).toBe(42);
  });

  it("parses default before type", () => {
    const def = parseParamString("body.name", "param(default:'machine-1', type:string, required:false)");
    expect(def.default).toBe("machine-1");
    expect(def.type).toBe("string");
  });

  it("skips static leaves", () => {
    const defs = parseParams({
      message: "hello",
      name: "param(type:string, default:'a', required:false)",
    });
    expect(defs).toHaveLength(1);
    expect(defs[0]?.path).toBe("name");
  });

  it("wraps path on parse errors", () => {
    expect(() =>
      parseParams({
        bad: "param(type:string, title:'bad,comma')",
      }),
    ).toThrow(/bad:/);
  });

  it("stops at the first invalid param", () => {
    expect(() =>
      parseParams({
        first: "param(type:string, title:'bad,comma')",
        second: "param(type:string, default:'ok')",
      }),
    ).toThrow(/first:/);
  });
});
