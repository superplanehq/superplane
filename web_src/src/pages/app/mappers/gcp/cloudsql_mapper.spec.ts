import { describe, expect, it } from "vitest";
import { createDatabaseMapper, getDatabaseMapper, deleteDatabaseMapper } from "./cloudsql_mapper";
import { buildDetailsCtx, buildOutput } from "./vm_mapper_test_helpers";

describe("cloudsql mappers getExecutionDetails", () => {
  it("createDatabase surfaces the created database details", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [buildOutput({ name: "app_db", instance: "my-instance", charset: "UTF8", collation: "en_US.UTF8" })],
        },
      },
    });
    const details = createDatabaseMapper.getExecutionDetails(ctx);
    expect(details["Database"]).toBe("app_db");
    expect(details["Instance"]).toBe("my-instance");
    expect(details["Charset"]).toBe("UTF8");
    expect(details["Completed At"]).toBeDefined();
  });

  it("getDatabase surfaces the fetched database details", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput({ name: "app_db", instance: "my-instance" })] } },
    });
    const details = getDatabaseMapper.getExecutionDetails(ctx);
    expect(details["Database"]).toBe("app_db");
    expect(details["Instance"]).toBe("my-instance");
  });

  it("deleteDatabase marks the database as deleted", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput({ name: "app_db", instance: "my-instance", deleted: true })] } },
    });
    const details = deleteDatabaseMapper.getExecutionDetails(ctx);
    expect(details["Database"]).toBe("app_db");
    expect(details["Deleted"]).toBe("true");
  });

  it("does not throw when outputs are missing", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => getDatabaseMapper.getExecutionDetails(ctx)).not.toThrow();
  });
});
