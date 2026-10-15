/* eslint-disable react-refresh/only-export-components -- shared test renderer and telemetry fixture */
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";
import { LiveHeaderSpendProvider, useReportLiveHeaderSpend } from "../liveHeaderSpendContext";
import type { SPLIT_RUN_RUNNING } from "../splitRunMocks";
import { AutomationsConsoleVariant } from "./AutomationsConsoleVariant";

type ConsoleProps = Parameters<typeof AutomationsConsoleVariant>[0];

export function renderConsole(
  fixture: typeof SPLIT_RUN_RUNNING,
  extra: Partial<ConsoleProps> = {},
  liveSpend?: [string, number, number],
) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const ui = (next: typeof fixture) => (
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <ThemeProvider>
          <TooltipProvider>
            <LiveHeaderSpendProvider>
              {liveSpend ? <ReportFooterSpend spend={liveSpend} /> : null}
              <AutomationsConsoleVariant fixture={next} source={next.source} {...extra} />
            </LiveHeaderSpendProvider>
          </TooltipProvider>
        </ThemeProvider>
      </MemoryRouter>
    </QueryClientProvider>
  );
  const view = render(ui(fixture));
  return { ...view, rerenderConsole: (next: typeof fixture) => view.rerender(ui(next)) };
}

function ReportFooterSpend({ spend }: { spend: [string, number, number] }) {
  useReportLiveHeaderSpend(...spend);
  return null;
}
