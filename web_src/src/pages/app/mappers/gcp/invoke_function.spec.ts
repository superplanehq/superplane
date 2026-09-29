import { describe, expect, it } from "bun:test";

import { invokeFunctionMapper } from "./invoke_function";
import { buildDetailsCtx, buildOutput } from "./vm_mapper_test_helpers";

describe("invokeFunctionMapper.getExecutionDetails", () => {
  it("does not throw when outputs are missing", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => invokeFunctionMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("maps functionName, executionId, and resultRaw", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            buildOutput({
              functionName: "projects/p/locations/us/functions/hello",
              executionId: "exec-9",
              resultRaw: "ok",
            }),
          ],
        },
      },
    });
    const details = invokeFunctionMapper.getExecutionDetails(ctx);
    expect(details["Function"]).toBe("hello");
    expect(details["Execution ID"]).toBe("exec-9");
    expect(details["Result"]).toBe("ok");
  });

  it("stringifies a non-string result when resultRaw is absent", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [buildOutput({ result: { status: "done" } })],
        },
      },
    });
    const details = invokeFunctionMapper.getExecutionDetails(ctx);
    expect(details["Result"]).toBe('{"status":"done"}');
  });
});
