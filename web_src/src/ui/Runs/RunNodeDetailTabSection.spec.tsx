import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { ThemeProvider } from "@/contexts/ThemeProvider";

import { RunNodeDetailTabSection } from "./RunNodeDetailTabSection";

describe("RunNodeDetailTabSection", () => {
  it("shows the whole payload string on the Payload tab, not a truncated preview", () => {
    // A failed node's payload is one long error string. It must stay readable
    // here without opening the copy button or the database.
    const longError =
      "error executing request: dial tcp 10.0.0.1:443 connect: connection blocked by policy rule policy-1234567890-abcdefghij";

    const { container } = render(
      <RunNodeDetailTabSection
        activeTab="payload"
        tabData={{ payload: { error: longError } }}
        hasDetailsSection={false}
        hasPayload
        hasConfig={false}
        headerEventBadge={null}
        onSelectTab={() => {}}
      />,
      { wrapper: ThemeProvider },
    );

    expect(container.textContent).toContain(longError);
    expect(container.textContent).not.toContain("...");
  });
});
