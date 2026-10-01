import { describe, expect, it } from "bun:test";

import { isMonacoWorkerAsset, monacoChunkFileName, monacoWorkerAppOriginRuntime } from "./monacoAssetUrls";

describe("Monaco worker asset URLs", () => {
  it.each(["editor.worker", "json.worker", "css.worker", "html.worker", "ts.worker"])(
    "serves a bundled %s file from the page origin",
    (worker) => {
      const filename = `assets/${worker}-Bx123.js`;

      expect(isMonacoWorkerAsset(filename)).toBe(true);
      expect(monacoWorkerAppOriginRuntime(filename)).toBe(
        `new URL("/assets/${worker}-Bx123.js", self.location.origin).href`,
      );
    },
  );

  it("leaves application assets on the configured asset base", () => {
    expect(isMonacoWorkerAsset("assets/index-Bx123.js")).toBe(false);
    expect(monacoWorkerAppOriginRuntime("assets/index-Bx123.js")).toBeUndefined();
    expect(monacoWorkerAppOriginRuntime("assets/monaco-editor-Bx123.js")).toBeUndefined();
  });

  it("does not keep the CDN release prefix in the worker path", () => {
    const runtime = monacoWorkerAppOriginRuntime("assets/editor.worker-Bx123.js");

    expect(runtime).not.toContain("assets.superplane.com");
    expect(runtime).not.toContain("/releases/");
  });
});

describe("Monaco chunk names", () => {
  it("keeps the shared monaco-editor chunk recognizable", () => {
    expect(
      monacoChunkFileName({
        name: "monaco-editor",
        facadeModuleId: null,
        moduleIds: ["\0rolldown/runtime.js", "/app/node_modules/monaco-editor/esm/vs/editor/editor.main.js"],
      }),
    ).toBe("assets/[name]-[hash].js");
  });

  it("marks a bundled editor facade that does not include monaco-editor in its module name", () => {
    expect(
      monacoChunkFileName({
        name: "editor.main",
        facadeModuleId: "/app/node_modules/monaco-editor/esm/vs/editor/editor.main.js",
        moduleIds: [],
      }),
    ).toBe("assets/monaco-editor-[name]-[hash].js");
  });

  it("leaves a chunk that includes application code unnamed", () => {
    expect(
      monacoChunkFileName({
        name: "MonacoEditor",
        facadeModuleId: "/app/web_src/src/ui/MonacoEditor.tsx",
        moduleIds: [
          "/app/web_src/src/ui/MonacoEditor.tsx",
          "/app/node_modules/monaco-editor/esm/vs/editor/editor.main.js",
        ],
      }),
    ).toBe("assets/[name]-[hash].js");
  });
});
