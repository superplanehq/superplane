import { render, screen, waitFor } from "@testing-library/react";
import { createContext, useContext, type ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";

const TabsContext = createContext<{ value: string }>({ value: "latest" });

vi.mock("../CanvasPage", () => ({
  COMPONENT_SIDEBAR_WIDTH_STORAGE_KEY: "component-sidebar-width",
}));

vi.mock("@/components/ui/tabs", () => ({
  Tabs: ({ value, children }: { value: string; children?: ReactNode }) => (
    <TabsContext.Provider value={{ value }}>{children}</TabsContext.Provider>
  ),
  TabsContent: ({ value, children }: { value: string; children?: ReactNode }) => {
    const context = useContext(TabsContext);

    if (context.value !== value) {
      return null;
    }

    return <div data-testid={`tab-content-${value}`}>{children}</div>;
  },
}));

vi.mock("@/components/ui/button", () => ({
  Button: ({ children, ...props }: { children?: ReactNode }) => <button {...props}>{children}</button>,
}));

vi.mock("@/components/ui/loading-button", () => ({
  LoadingButton: ({ children, ...props }: { children?: ReactNode }) => <button {...props}>{children}</button>,
}));

vi.mock("@/components/ui/dialog", () => ({
  Dialog: ({ children }: { children?: ReactNode }) => <>{children}</>,
  DialogContent: ({ children }: { children?: ReactNode }) => <div>{children}</div>,
  DialogFooter: ({ children }: { children?: ReactNode }) => <div>{children}</div>,
  DialogHeader: ({ children }: { children?: ReactNode }) => <div>{children}</div>,
  DialogTitle: ({ children }: { children?: ReactNode }) => <div>{children}</div>,
}));

vi.mock("@/components/ui/input", () => ({
  Input: (props: object) => <input {...props} />,
}));

vi.mock("@/components/ui/label", () => ({
  Label: ({ children }: { children?: ReactNode }) => <label>{children}</label>,
}));

vi.mock("@/lib/integrationDisplayName", () => ({
  getIntegrationTypeDisplayName: () => "Integration",
}));

vi.mock("@/lib/utils", () => ({
  resolveIcon: () => () => <div data-testid="resolved-icon" />,
}));

vi.mock("@/ui/componentSidebar/integrationIcons", () => ({
  getHeaderIconSrc: () => undefined,
  IntegrationIcon: () => <div data-testid="integration-icon" />,
}));

vi.mock("@/hooks/useIntegrations", () => ({
  useAvailableIntegrations: () => ({ data: [] }),
  useCreateIntegration: () => ({
    mutateAsync: vi.fn(),
    reset: vi.fn(),
    isPending: false,
  }),
  useIntegration: () => ({
    data: undefined,
    isLoading: false,
  }),
  useUpdateIntegration: () => ({
    mutateAsync: vi.fn(),
    reset: vi.fn(),
    isPending: false,
  }),
}));

vi.mock("@/ui/configurationFieldRenderer", () => ({
  ConfigurationFieldRenderer: () => <div data-testid="configuration-field-renderer" />,
}));

vi.mock("@/lib/errors", () => ({
  getApiErrorMessage: () => "error",
}));

vi.mock("@/lib/toast", () => ({
  showErrorToast: vi.fn(),
}));

vi.mock("@/ui/IntegrationCreateDialog", () => ({
  IntegrationCreateDialog: () => null,
}));

vi.mock("@/ui/IntegrationInstructions", () => ({
  IntegrationInstructions: () => null,
}));

vi.mock("./DocsTab", () => ({
  DocsTab: () => <div data-testid="docs-tab">Docs Tab</div>,
}));

vi.mock("./LatestTab", () => ({
  LatestTab: () => <div data-testid="latest-tab">Latest Tab</div>,
}));

vi.mock("./SettingsTab", () => ({
  SettingsTab: () => <div data-testid="settings-tab">Settings Tab</div>,
}));

vi.mock("./pages", () => ({
  ExecutionChainPage: () => <div data-testid="execution-chain-page" />,
  HistoryQueuePage: () => <div data-testid="history-queue-page" />,
  PageHeader: () => <div data-testid="page-header" />,
}));

vi.mock("@/pages/workflowv2/utils", () => ({
  mapTriggerEventToSidebarEvent: vi.fn(),
}));

import { ComponentSidebar } from "./index";

function renderSidebar(props?: Partial<React.ComponentProps<typeof ComponentSidebar>>) {
  return render(
    <ComponentSidebar
      isOpen={true}
      canvasMode="live"
      latestEvents={[]}
      nextInQueueEvents={[]}
      totalInQueueCount={0}
      totalInHistoryCount={0}
      showSettingsTab={true}
      nodeName="Node"
      nodeConfiguration={{}}
      nodeConfigurationFields={[]}
      workflowNodes={[]}
      actions={[]}
      triggers={[]}
      {...props}
    />,
  );
}

describe("ComponentSidebar", () => {
  it("shows runs content in live mode", () => {
    renderSidebar({
      canvasMode: "live",
      currentTab: "latest",
    });

    expect(screen.getByText("Runs")).toBeInTheDocument();
    expect(screen.getByTestId("latest-tab")).toBeInTheDocument();
    expect(screen.queryByTestId("settings-tab")).not.toBeInTheDocument();
  });

  it("hides runs content in edit mode and normalizes latest tab to settings", async () => {
    const onTabChange = vi.fn();

    renderSidebar({
      canvasMode: "edit",
      currentTab: "latest",
      onTabChange,
    });

    expect(screen.queryByText("Runs")).not.toBeInTheDocument();
    expect(screen.queryByTestId("latest-tab")).not.toBeInTheDocument();
    expect(screen.getByTestId("settings-tab")).toBeInTheDocument();

    await waitFor(() => {
      expect(onTabChange).toHaveBeenCalledWith("settings");
    });
  });
});
