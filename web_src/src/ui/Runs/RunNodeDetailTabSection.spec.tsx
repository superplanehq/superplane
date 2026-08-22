import { render, screen } from "@testing-library/react";
import type { ReactElement } from "react";
import { describe, expect, it } from "vitest";
import { ThemeContext, type ThemeContextType } from "@/contexts/themeContextState";
import { RunNodeDetailTabSection } from "./RunNodeDetailTabSection";
import type { RunNodeDetailTabData } from "./types";

const themeValue: ThemeContextType = {
  preference: "light",
  resolvedTheme: "light",
  setPreference: () => {},
};

function renderWithTheme(ui: ReactElement) {
  return render(<ThemeContext.Provider value={themeValue}>{ui}</ThemeContext.Provider>);
}

describe("RunNodeDetailTabSection", () => {
  it("renders long payload string values in full instead of truncating them", () => {
    // Quote-free so the JSON-escaped rendering matches the raw value exactly.
    const longError =
      "error executing request: dial tcp 10.0.0.1:443 connect: connection blocked by policy rule policy-1234567890-abcdefghij";
    const tabData = { payload: { error: longError } } as unknown as RunNodeDetailTabData;

    renderWithTheme(
      <RunNodeDetailTabSection
        activeTab="payload"
        tabData={tabData}
        hasDetailsSection={false}
        hasPayload
        hasConfig={false}
        headerEventBadge={null}
        onSelectTab={() => {}}
      />,
    );

    // The default @uiw/react-json-view truncates string values after 30 chars
    // with an ellipsis, dropping the tail of the error. Assert the whole value
    // is present so the failure message stays readable in the inspector.
    expect(screen.getByText(longError)).toBeInTheDocument();
  });
});
