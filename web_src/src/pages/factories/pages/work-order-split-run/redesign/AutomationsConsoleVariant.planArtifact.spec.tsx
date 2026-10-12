import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it } from "bun:test";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";

import { LiveHeaderSpendProvider } from "../liveHeaderSpendContext";
import { SPLIT_RUN_RUNNING } from "../splitRunMocks";
import { AutomationsConsoleVariant } from "./AutomationsConsoleVariant";

type ConsoleProps = Parameters<typeof AutomationsConsoleVariant>[0];

function renderConsole(fixture: typeof SPLIT_RUN_RUNNING, extra: Partial<ConsoleProps> = {}) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <ThemeProvider>
          <TooltipProvider>
            <LiveHeaderSpendProvider>
              <AutomationsConsoleVariant fixture={fixture} source={fixture.source} {...extra} />
            </LiveHeaderSpendProvider>
          </TooltipProvider>
        </ThemeProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("AutomationsConsoleVariant plan artifacts", () => {
  it("shows a stored spec.md planning row as plan.md", () => {
    renderConsole(SPLIT_RUN_RUNNING, {
      artifacts: [
        {
          id: "art-old-plan",
          key: "spec:11111111-1111-4111-8111-111111111111",
          type: "TYPE_MARKDOWN",
          data: { name: "spec.md", title: "spec.md", body: "# Old plan" },
        },
        {
          id: "art-user-spec",
          type: "TYPE_MARKDOWN",
          data: { name: "spec.md", title: "spec.md", body: "# User file" },
        },
      ],
    });

    const text = screen.getByTestId("redesign-console-artifacts").textContent ?? "";
    expect(text).toContain("plan.md");
    expect(text).toContain("spec.md");
    expect(text.indexOf("plan.md")).toBeLessThan(text.indexOf("spec.md"));
  });
});
