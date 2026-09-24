import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "bun:test";
import { SettingsTab } from "./SettingsTab";

vi.mock("./configurationView/ConfigurationView", () => ({
  ConfigurationView: () => <div data-testid="configuration-view" />,
}));

vi.mock("@/hooks/useFactoryData", () => ({
  useFactories: () => ({ data: [] }),
}));

describe("SettingsTab", () => {
  it("renders customField in read-only mode", () => {
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <MemoryRouter>
          <SettingsTab
            mode="edit"
            nodeName="Wait node"
            configuration={{}}
            configurationFields={[]}
            onSave={vi.fn()}
            readOnly
            customField={() => <div data-testid="custom-field">Wait controls</div>}
          />
        </MemoryRouter>
      </QueryClientProvider>,
    );

    expect(screen.getByTestId("configuration-view")).toBeInTheDocument();
    expect(screen.getByTestId("custom-field")).toBeInTheDocument();
    expect(screen.queryByTestId("save-node-button")).not.toBeInTheDocument();
  });
});
