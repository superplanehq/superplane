import MonacoReactEditor from "@monaco-editor/react";
import type { EditorProps } from "@monaco-editor/react";
import { lazy, Suspense } from "react";

export type { EditorProps, Monaco } from "@monaco-editor/react";

const ConfiguredEditor = lazy(async () => {
  const { setupMonaco } = await import("@/lib/setupMonaco");
  await setupMonaco();
  return { default: MonacoReactEditor };
});

export function Editor(props: EditorProps) {
  if (import.meta.env.MODE === "test") {
    return <MonacoReactEditor {...props} />;
  }

  return (
    <Suspense fallback={<EditorLoading {...props} />}>
      <ConfiguredEditor {...props} />
    </Suspense>
  );
}

function EditorLoading({ width = "100%", height = "100%", loading = "Loading...", wrapperProps = {} }: EditorProps) {
  return (
    <section style={{ display: "flex", position: "relative", textAlign: "initial", width, height }} {...wrapperProps}>
      <div style={{ display: "flex", height: "100%", width: "100%", justifyContent: "center", alignItems: "center" }}>
        {loading}
      </div>
    </section>
  );
}

export default Editor;
