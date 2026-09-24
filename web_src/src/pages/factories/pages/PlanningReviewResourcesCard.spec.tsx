import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { FEATURE_WORKSPACE_MCP, FEATURE_WORKSPACE_SKILLS } from "@/lib/experimentalFeatures";
import { HEADER_MCP_RESOURCE, INLINE_SKILL } from "../__fixtures__/agentResourceFixtures";
import { PRIMARY_FACTORY_ID, PRIMARY_FACTORY_KEY } from "../__fixtures__/factoryPageResponses";
import { disabledAgentResourceIds } from "./disabledAgentResourceIds";
import { PlanningReviewResourcesCard } from "./PlanningReviewResourcesCard";

const useFactoryAgentResources = vi.hoisted(() => vi.fn());
const useFactoryAgentResourceTools = vi.hoisted(() => vi.fn());
const useExperimentalFeature = vi.hoisted(() => vi.fn());

vi.mock("@/hooks/useFactoryAgentResources", () => ({
  useFactoryAgentResources,
  useFactoryAgentResourceTools,
}));

vi.mock("@/hooks/useExperimentalFeature", () => ({
  useExperimentalFeature,
}));

function renderCard(
  props: Partial<Parameters<typeof PlanningReviewResourcesCard>[0]> & {
    onDisabledIdsChange?: (ids: string[]) => void;
  } = {},
) {
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <PlanningReviewResourcesCard
          organizationId="org-1"
          factoryId={PRIMARY_FACTORY_ID}
          factoryKey={PRIMARY_FACTORY_KEY}
          disabledIds={[]}
          onDisabledIdsChange={props.onDisabledIdsChange ?? vi.fn()}
          {...props}
        />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("PlanningReviewResourcesCard", () => {
  beforeEach(() => {
    useExperimentalFeature.mockReturnValue({
      has: (feature: string) => feature === FEATURE_WORKSPACE_MCP || feature === FEATURE_WORKSPACE_SKILLS,
      enabledExperimentalFeatures: [FEATURE_WORKSPACE_MCP, FEATURE_WORKSPACE_SKILLS],
      isLoading: false,
    });
    useFactoryAgentResourceTools.mockReturnValue({ data: [], isLoading: false, isError: false });
    useFactoryAgentResources.mockImplementation((_org: string, _factory: string, kind: string) => {
      if (kind === "KIND_SKILL") {
        return { data: [INLINE_SKILL], isLoading: false, isError: false };
      }
      return { data: [HEADER_MCP_RESOURCE], isLoading: false, isError: false };
    });
  });

  it("hides when the feature is off", () => {
    useExperimentalFeature.mockReturnValue({
      has: () => false,
      enabledExperimentalFeatures: [],
      isLoading: false,
    });
    renderCard();
    expect(screen.queryByTestId("planning-review-resources")).not.toBeInTheDocument();
  });

  it("shows an empty callout with a settings link", () => {
    useFactoryAgentResources.mockReturnValue({ data: [], isLoading: false, isError: false });
    renderCard();

    expect(screen.getByTestId("planning-review-resources-empty-mcp")).toHaveTextContent(
      "Add MCP servers on the workspace settings page.",
    );
    expect(screen.getAllByTestId("planning-review-resources-settings")[0]).toHaveAttribute(
      "href",
      `/org-1/workspaces/${PRIMARY_FACTORY_KEY.toLowerCase()}/settings/workspace/mcp`,
    );
  });

  it("lists MCP servers and skills with a manage link", async () => {
    const user = userEvent.setup();
    const onDisabledIdsChange = vi.fn();
    renderCard({ onDisabledIdsChange });

    expect(screen.getByText("docs")).toBeInTheDocument();
    expect(screen.getByText("MCP server")).toBeInTheDocument();
    expect(screen.getByText("review-copy")).toBeInTheDocument();
    expect(screen.getByText("Skill")).toBeInTheDocument();
    expect(screen.getAllByTestId("planning-review-resources-manage")[0]).toHaveAttribute(
      "href",
      `/org-1/workspaces/${PRIMARY_FACTORY_KEY.toLowerCase()}/settings/workspace/mcp`,
    );

    await user.click(screen.getByTestId(`planning-review-resource-${HEADER_MCP_RESOURCE.id}`));
    expect(onDisabledIdsChange).toHaveBeenCalledWith([HEADER_MCP_RESOURCE.id]);
  });

  it("shows the enabled tool count on the collapsed MCP row", () => {
    useFactoryAgentResourceTools.mockReturnValue({
      data: [
        { name: "search", readOnly: true },
        { name: "list_issues", readOnly: true },
        { name: "create_issue", readOnly: false },
        { name: "update_issue", readOnly: false },
      ],
      isLoading: false,
      isError: false,
    });
    renderCard({
      disabledTools: { [HEADER_MCP_RESOURCE.id ?? ""]: ["create_issue"] },
    });

    expect(screen.getByTestId(`mcp-tools-count-${HEADER_MCP_RESOURCE.id}`)).toHaveTextContent("3/4");
  });

  it("keeps a workspace-disabled resource listed and locked", () => {
    useFactoryAgentResources.mockImplementation((_org: string, _factory: string, kind: string) => {
      if (kind === "KIND_SKILL") {
        return { data: [], isLoading: false, isError: false };
      }
      return { data: [{ ...HEADER_MCP_RESOURCE, enabled: false }], isLoading: false, isError: false };
    });
    renderCard();

    expect(screen.getByText("Off for the workspace.")).toBeInTheDocument();
    expect(screen.getByTestId(`planning-review-resource-${HEADER_MCP_RESOURCE.id}`)).toBeDisabled();
  });
});

describe("disabledAgentResourceIds", () => {
  it("reads string ids and ignores other values", () => {
    expect(disabledAgentResourceIds(undefined)).toEqual([]);
    expect(
      disabledAgentResourceIds({
        disabledAgentResourceIds: [HEADER_MCP_RESOURCE.id ?? "", 1, ""],
      }),
    ).toEqual([HEADER_MCP_RESOURCE.id]);
  });
});
