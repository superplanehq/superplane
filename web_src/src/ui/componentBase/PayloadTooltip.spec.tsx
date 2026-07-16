import { render, screen } from "@testing-library/react";
import React from "react";
import { describe, expect, it, vi } from "vitest";
import { PayloadTooltip } from "./PayloadTooltip";

const monacoThemes = vi.hoisted((): string[] => []);

vi.mock("@/contexts/useTheme", () => ({
  useTheme: () => ({ preference: "dark", resolvedTheme: "dark", setPreference: () => undefined }),
}));

vi.mock("@monaco-editor/react", () => ({
  default: ({ theme, value }: { theme?: string; value?: string }) => {
    monacoThemes.push(theme ?? "");
    return <pre data-testid="monaco-editor">{value}</pre>;
  },
}));

vi.mock("@tippyjs/react/headless", () => ({
  default: ({
    children,
    render: renderTooltip,
  }: {
    children: React.ReactNode;
    render: (attrs: Record<string, unknown>) => React.ReactNode;
  }) => (
    <div>
      {renderTooltip({})}
      {children}
    </div>
  ),
}));

describe("PayloadTooltip", () => {
  it("uses dark tooltip surfaces and the dark Monaco theme", () => {
    render(
      <PayloadTooltip title="Payload" value={{ ok: true }}>
        <span>payload trigger</span>
      </PayloadTooltip>,
    );

    expect(screen.getByText("Payload").parentElement?.parentElement?.className).toContain("dark:bg-gray-900");
    expect(screen.getByText("Payload").className).toContain("dark:text-gray-400");
    expect(monacoThemes).toContain("vs-dark");
  });

  it("renders missing text payloads without crashing", () => {
    render(
      <PayloadTooltip title="Payload" value={undefined} contentType="text">
        <span>payload trigger</span>
      </PayloadTooltip>,
    );

    expect(screen.getByTestId("monaco-editor")).toHaveTextContent("");
  });
});
