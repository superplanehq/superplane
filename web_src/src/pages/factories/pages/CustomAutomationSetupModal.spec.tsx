import { act, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { load } from "js-yaml";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { BrowserRouter, Link, MemoryRouter, useLocation, useNavigate } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";
import { useState, type ReactNode } from "react";
import type { CanvasesCanvasRun, FactoriesFactoryCustomAutomation } from "@/api-client";
import type { CustomAutomationColumn } from "../lib/customAutomation";
import { useCommandPaletteShortcuts } from "@/components/GlobalCommandPalette/hooks";
import { ThemeProvider } from "@/contexts/ThemeProvider";
import { FactoriesLayoutContext } from "../layout/factoriesLayoutContext";
import { CustomAutomationSetupModal } from "./CustomAutomationSetupModal";

const hooks = vi.hoisted(() => ({
  describe: vi.fn(),
  create: vi.fn(),
  update: vi.fn(),
  delete: vi.fn(),
  runs: vi.fn(),
  run: vi.fn(),
  runtime: vi.fn(),
}));
vi.mock("@/hooks/useFactoryCustomAutomationData", () => ({
  factoryCustomAutomationKey: (...parts: string[]) => ["custom", ...parts],
  useFactoryCustomAutomation: (...args: unknown[]) => hooks.describe(...args) ?? {},
  useCreateFactoryCustomAutomation: () => ({ mutateAsync: hooks.create }),
  useUpdateFactoryCustomAutomation: () => ({ mutateAsync: hooks.update }),
}));
vi.mock("@/hooks/useFactoryData", () => ({
  factoryAppsKey: (...parts: string[]) => ["factory-apps", ...parts],
  useDeleteFactoryAutomation: () => ({ mutateAsync: hooks.delete }),
}));
vi.mock("@/hooks/useCanvasData", () => ({
  useInfiniteCanvasRuns: (...args: unknown[]) => hooks.runs(...args),
  useDescribeRun: (...args: unknown[]) => hooks.run(...args),
  useEventExecutions: () => ({ data: { executions: [] } }),
  useCanvasVersion: () => ({ data: { spec: { nodes: [] } } }),
}));
vi.mock("@/hooks/useCanvasWebsocket", () => ({ useCanvasRuntimeWebsocket: hooks.runtime }));
vi.mock("@/hooks/useSecrets", () => ({ useSecrets: () => ({ data: [] }) }));
// Agent model controls have separate coverage; retain the real setup form and step selection.
vi.mock("@/hooks/useSelectableLLMModels", () => ({ useSelectableLLMModels: () => ({}) }));
vi.mock("./PlanningReviewForm", () => ({ PlanningReviewForm: () => null }));

const automation: FactoriesFactoryCustomAutomation = {
  automation: { id: "custom-1", name: "Deploy preview" },
  settings: {
    startMode: "START_MODE_MANUAL",
    steps: [{ name: "Deploy", type: "TYPE_PROMPT", prompt: "Deploy the preview." }],
  },
};
const run: CanvasesCanvasRun = {
  id: "run-1",
  canvasId: "custom-1",
  state: "STATE_FINISHED",
  result: "RESULT_PASSED",
  task: { id: "task-1", key: "SP-123", title: "Preview settings" },
};

const createApp = vi.fn(async () => {});

function BrowserControls() {
  useCommandPaletteShortcuts({
    createCanvas: createApp,
    createCanvasDisabled: false,
    enabled: true,
    open: false,
    page: "root",
    search: "",
    setOpen: () => {},
    setPage: () => {},
    setSearch: () => {},
  });
  const location = useLocation();
  const navigate = useNavigate();
  return (
    <>
      <output data-testid="url">{location.search}</output>
      <button onClick={() => navigate(-1)}>Browser Back</button>
      <button onClick={() => navigate(1)}>Browser Forward</button>
    </>
  );
}

function renderModal(
  search = "?customAutomation=custom-1",
  automationId: string | undefined = search.includes("customAutomation=new") ? undefined : "custom-1",
  columnKey?: CustomAutomationColumn,
  yamlEnabled = false,
) {
  const onClose = vi.fn();
  render(
    <MemoryRouter initialEntries={[`/org/workspaces/sp/lines/line${search}`]}>
      <BrowserControls />
      <AutomationModalProviders>
        <CustomAutomationSetupModal
          lineId="line"
          automationId={automationId}
          columnKey={columnKey}
          yamlEnabled={yamlEnabled}
          onClose={onClose}
        />
      </AutomationModalProviders>
    </MemoryRouter>,
  );
  return onClose;
}

function AutomationModalProviders({ children }: { children: ReactNode }) {
  const [client] = useState(() => new QueryClient({ defaultOptions: { queries: { retry: false } } }));
  return (
    <FactoriesLayoutContext.Provider
      value={{
        organizationId: "org",
        factoryId: "factory",
        factoryKey: "sp",
        factory: null,
        factories: [],
        openCreateWorkOrder: vi.fn(),
      }}
    >
      <QueryClientProvider client={client}>
        <ThemeProvider>{children}</ThemeProvider>
      </QueryClientProvider>
    </FactoriesLayoutContext.Provider>
  );
}

describe("custom automation configuration run history", () => {
  beforeEach(() => {
    Object.values(hooks).forEach((hook) => hook.mockReset());
    hooks.describe.mockReturnValue({ data: automation });
    hooks.runs.mockImplementation((canvasId: string) => ({
      data: {
        pages: [
          {
            runs:
              canvasId === "custom-1" ? [run] : [{ ...run, id: "other-run", task: { title: "Other automation task" } }],
          },
        ],
      },
    }));
    hooks.run.mockReturnValue({ data: { run } });
  });

  it.each(["START_MODE_MANUAL", "START_MODE_AUTOMATIC"] as const)(
    "shows only the saved automation's runs in %s mode and restores run navigation",
    async (startMode) => {
      hooks.describe.mockReturnValue({ data: { ...automation, settings: { ...automation.settings, startMode } } });
      const onClose = renderModal();
      const dialog = within(screen.getByRole("dialog"));
      expect(dialog.getByRole("tab", { name: "Configuration" })).toHaveAttribute("aria-selected", "true");
      expect(hooks.runs).not.toHaveBeenCalled();
      const user = userEvent.setup();
      await user.click(dialog.getByRole("tab", { name: "Runs" }));
      expect(dialog.getByText("SP-123")).toBeVisible();
      expect(dialog.getByText("Preview settings")).toBeVisible();
      expect(dialog.queryByText("Other automation task")).not.toBeInTheDocument();
      expect(hooks.runs).toHaveBeenCalledWith("custom-1", { includeTask: true });
      expect(hooks.runtime).toHaveBeenCalledWith("custom-1", "org", true);
      await user.click(dialog.getByRole("button", { name: "Open run run-1" }));
      expect(screen.getByTestId("url")).toHaveTextContent("customAutomation=custom-1");
      expect(screen.getByTestId("url")).toHaveTextContent("automationRun=run-1");
      expect(dialog.getByRole("button", { name: "Run history" })).toBeVisible();
      fireEvent.click(screen.getByRole("button", { name: "Browser Back", hidden: true }));
      await waitFor(() => expect(dialog.getByRole("table")).toBeVisible());
      fireEvent.click(screen.getByRole("button", { name: "Browser Back", hidden: true }));
      await waitFor(() =>
        expect(dialog.getByRole("tab", { name: "Configuration" })).toHaveAttribute("aria-selected", "true"),
      );
      fireEvent.click(screen.getByRole("button", { name: "Browser Forward", hidden: true }));
      await waitFor(() => expect(dialog.getByRole("table")).toBeVisible());
      fireEvent.click(screen.getByRole("button", { name: "Browser Forward", hidden: true }));
      await waitFor(() => expect(dialog.getByRole("button", { name: "Run history" })).toBeVisible());
      await user.click(dialog.getByRole("button", { name: "Run history" }));
      expect(dialog.getByRole("table")).toBeVisible();
      expect(screen.getByTestId("url")).not.toHaveTextContent("automationRun=");
      expect(onClose).not.toHaveBeenCalled();
    },
  );

  it.each(["automationTab=runs", "automationRun=run-1"])("opens explicit history links: %s", (selection) => {
    renderModal(`?customAutomation=custom-1&${selection}`);
    expect(screen.getByRole("tab", { name: "Runs" })).toHaveAttribute("aria-selected", "true");
    expect(
      selection.includes("automationRun")
        ? screen.getByRole("button", { name: "Run history" })
        : screen.getByRole("table"),
    ).toBeVisible();
  });

  it("retains unsaved fields and the selected step when switching tabs", async () => {
    renderModal();
    const user = userEvent.setup();
    await user.clear(screen.getByLabelText("Automation name"));
    await user.type(screen.getByLabelText("Automation name"), "Draft automation");
    await user.click(screen.getByRole("button", { name: /Deploy.*Deploy the preview/ }));
    await user.clear(screen.getByLabelText("Name"));
    await user.type(screen.getByLabelText("Name"), "Draft step");
    await user.click(screen.getByRole("tab", { name: "Runs" }));
    expect(screen.queryByRole("button", { name: "Save" })).not.toBeInTheDocument();
    await user.click(screen.getByRole("tab", { name: "Configuration" }));
    expect(screen.getByLabelText("Name")).toHaveValue("Draft step");
    await user.click(screen.getByRole("button", { name: /Manual.*Runs when/ }));
    expect(screen.getByLabelText("Automation name")).toHaveValue("Draft automation");
    expect(hooks.update).not.toHaveBeenCalled();
  });

  it.each([
    ["backlog", "Backlog"],
    ["implement", "Implement"],
    ["verify", "Verify"],
    ["done", "Done"],
  ] as const)("offers a leave trigger on %s", async (columnKey, label) => {
    hooks.describe.mockReturnValue({});
    renderModal("?customAutomation=new", undefined, columnKey);
    await userEvent.setup().click(screen.getByRole("button", { name: "Add trigger" }));
    expect(screen.getByRole("menuitem", { name: new RegExp(`When a task leaves ${label}`) })).toBeEnabled();
  });

  it.each([
    ["backlog", "When a task enters Backlog", "Manual"],
    ["implement", "When a task enters Implement", "Manual"],
    ["done", "When a task enters Done", "Manual"],
    ["verify", "Manual", "Every pull request commit"],
  ] as const)("starts a new %s automation on %s", (columnKey, label, absent) => {
    hooks.describe.mockReturnValue({});
    renderModal("?customAutomation=new", undefined, columnKey);
    expect(screen.getAllByText(label).length).toBeGreaterThan(0);
    expect(screen.queryByText(absent)).not.toBeInTheDocument();
  });

  it("keeps new automation configuration available without mounting history", async () => {
    hooks.describe.mockReturnValue({});
    const onClose = renderModal("?customAutomation=new&automationTab=runs&automationRun=run-1");
    expect(screen.queryByRole("tab", { name: "Runs" })).not.toBeInTheDocument();
    expect(screen.getByLabelText("Automation name")).toBeVisible();
    expect(screen.getByRole("button", { name: "Save" })).toBeVisible();
    expect(screen.queryByRole("button", { name: "Delete" })).not.toBeInTheDocument();
    expect(hooks.runs).not.toHaveBeenCalled();
    expect(hooks.runtime).not.toHaveBeenCalled();
    await userEvent.setup().click(screen.getByRole("button", { name: "Cancel" }));
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("saves custom-stage configuration through the shared editor with line-owned execution", async () => {
    hooks.describe.mockReturnValue({
      data: { ...automation, settings: { ...automation.settings, startMode: "START_MODE_AUTOMATIC" } },
    });
    const onClose = renderModal("?customAutomation=custom-1", "custom-1", "custom-qa");
    expect(screen.queryByRole("button", { name: "Add trigger" })).toBeNull();
    expect(screen.queryByTestId("custom-automation-runs-on-task")).toBeNull();
    await userEvent.setup().click(screen.getByRole("button", { name: "Save" }));
    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1));
    expect(hooks.update).toHaveBeenCalledWith(
      expect.objectContaining({
        name: "Deploy preview",
        settings: expect.objectContaining({ startMode: "START_MODE_AUTOMATIC", runsOnTask: true }),
      }),
    );
  });

  it("keeps deletion confirmation with configuration and closes after deletion", async () => {
    const onClose = renderModal();
    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: "Delete" }));
    expect(hooks.delete).not.toHaveBeenCalled();
    await user.click(screen.getByRole("button", { name: "Delete" }));
    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1));
    expect(hooks.delete).toHaveBeenCalledWith("custom-1");
  });
});

function mockAutomationSourceFetch(onAutomation: (method: string) => Response) {
  return vi.spyOn(globalThis, "fetch").mockImplementation(async (input, init) => {
    if (!String(input).includes("/api/v2/automations")) return Response.json({});
    return onAutomation(init?.method ?? "GET");
  });
}

describe("custom automation YAML draft", () => {
  const savedYaml = `# Saved from HTTP
apiVersion: v2
kind: Automation
name: Preview
agent:
  harness: codex
  model: gpt-5
  credentials: {source: integration, name: code-review}
skip_setup_commands: true
skip_repository_checkout: true
triggers:
  - use: task_enters_stage
steps:
  - name: Print marker
    bash: echo original
  - name: Review
    working_directory: repo
    prompt: Review the preview.
`;
  const savedAutomation = {
    id: "custom-1",
    factory_id: "factory",
    stage: "backlog",
    name: "Preview",
    yaml: savedYaml,
    revision: 1,
  };

  beforeEach(() => hooks.runs.mockReturnValue({ data: { pages: [] } }));
  afterEach(() => vi.restoreAllMocks());

  it("opens the form editor when the server verifies it can preserve the automation", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      Response.json(
        { message: "Open this automation in the form.", can_edit_in_form: true, live_version_id: "version-42" },
        { status: 409 },
      ),
    );
    hooks.describe.mockReturnValue({ data: automation, isFetchedAfterMount: true });
    hooks.update.mockResolvedValue(automation);
    const onClose = renderModal("?customAutomation=custom-1", "custom-1", "verify", true);
    const user = userEvent.setup();
    await user.click(await screen.findByRole("button", { name: "Edit in form" }));
    expect(screen.getByLabelText("Automation name")).toHaveValue("Deploy preview");
    await user.clear(screen.getByLabelText("Automation name"));
    await user.type(screen.getByLabelText("Automation name"), "Updated preview");
    await user.click(screen.getByRole("button", { name: "Save" }));
    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1));
    expect(hooks.update).toHaveBeenCalledWith(
      expect.objectContaining({ name: "Updated preview", expectedLiveVersionId: "version-42" }),
    );
  });

  it("protects a custom graph that the form cannot preserve", async () => {
    const conflict = Response.json({ message: "Open the developer canvas." }, { status: 409 });
    vi.spyOn(globalThis, "fetch").mockResolvedValue(conflict);
    renderModal("?customAutomation=custom-1", "custom-1", "verify", true);
    expect(await screen.findByRole("alert")).toBeVisible();
    expect(screen.queryByRole("button", { name: "Edit in form" })).not.toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Open full canvas" })).toBeVisible();
  });

  it("blocks the app creation shortcut only while the automation draft is dirty", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(Response.json(savedAutomation));
    createApp.mockClear();
    renderModal("?customAutomation=custom-1", "custom-1", "backlog", true);
    const name = await screen.findByLabelText("Automation name");
    const cancel = screen.getByRole("button", { name: "Cancel" });
    cancel.focus();
    fireEvent.keyDown(cancel, { key: "/", ctrlKey: true });
    expect(createApp).toHaveBeenCalledTimes(1);

    fireEvent.change(name, { target: { value: "Unsaved preview" } });
    cancel.focus();
    fireEvent.keyDown(cancel, { key: "/", ctrlKey: true });
    fireEvent.keyDown(cancel, { key: "/", metaKey: true });
    expect(createApp).toHaveBeenCalledTimes(1);
    expect(name).toHaveValue("Unsaved preview");

    await userEvent.setup().click(screen.getByRole("button", { name: "Edit YAML" }));
    fireEvent.change(screen.getByLabelText("Automation YAML"), { target: { value: savedYaml } });
    fireEvent.keyDown(screen.getByRole("button", { name: "Cancel" }), { key: "/", metaKey: true });
    expect(createApp).toHaveBeenCalledTimes(2);
  });

  it("opens the visual editor with an empty step list for a new automation", async () => {
    renderModal("?customAutomation=new", undefined, "verify", true);
    const user = userEvent.setup();
    expect(screen.queryByLabelText("Automation YAML")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Add step" })).toBeEnabled();
    expect(screen.queryAllByRole("button", { name: "Remove step" }).length).toBe(0);

    await user.click(screen.getByRole("button", { name: "Edit YAML" }));
    const yaml = screen.getByLabelText<HTMLTextAreaElement>("Automation YAML");
    expect(yaml.value).toMatch(/^steps: \[\]$/m);
    fireEvent.change(yaml, { target: { value: `${yaml.value}# Draft\n` } });
    await user.click(screen.getByRole("button", { name: "Open visual editor" }));
    expect(screen.getByRole("button", { name: "Add step" })).toBeEnabled();
  });

  it.each(["backlog", "custom-qa"] as const)(
    "round-trips visual and YAML edits on %s without changing identity",
    async (stage) => {
      const placed = { ...savedAutomation, stage };
      const fetch = vi
        .spyOn(globalThis, "fetch")
        .mockImplementation(async (_url, init) =>
          Response.json(init?.method === "PUT" ? { ...placed, revision: 2 } : placed),
        );
      const onClose = renderModal("?customAutomation=custom-1", "custom-1", stage, true);
      const user = userEvent.setup();
      expect(await screen.findByLabelText("Automation name")).toHaveValue("Preview");
      await user.click(screen.getByRole("button", { name: /Review.*Review the preview/ }));
      fireEvent.change(screen.getByLabelText("Name"), { target: { value: "Review changes" } });

      await user.click(screen.getByRole("button", { name: "Edit YAML" }));
      await user.click(screen.getByRole("button", { name: "Open visual editor" }));
      fireEvent.change(screen.getByLabelText("Automation name"), { target: { value: "Board preview" } });
      await user.click(screen.getByRole("button", { name: "Edit YAML" }));
      const projected = (screen.getByLabelText("Automation YAML") as HTMLTextAreaElement).value;
      expect(load(projected)).toMatchObject({
        name: "Board preview",
        agent: { harness: "codex", model: "gpt-5", credentials: { source: "integration", name: "code-review" } },
        skip_setup_commands: true,
        skip_repository_checkout: true,
        steps: [{ name: "Print marker" }, { name: "Review changes", working_directory: "repo" }],
      });
      const edited = `# Unsaved comment\n${projected.replace("echo original", "echo edited")}`;
      fireEvent.change(screen.getByLabelText("Automation YAML"), { target: { value: edited } });
      await user.click(screen.getByRole("button", { name: "Open visual editor" }));
      expect(screen.getByRole("button", { name: /Print marker.*echo edited/ })).toBeVisible();
      await user.click(screen.getByRole("button", { name: "Edit YAML" }));

      await user.click(screen.getByRole("button", { name: "Save" }));
      await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1));
      expect(fetch).toHaveBeenLastCalledWith(
        "/api/v2/automations/custom-1",
        expect.objectContaining({ method: "PUT", body: JSON.stringify({ yaml: edited }) }),
      );
    },
  );

  it("keeps invalid YAML editable and confirms before discarding changes", async () => {
    const fetch = mockAutomationSourceFetch(() => Response.json(savedAutomation));
    const onClose = renderModal("?customAutomation=custom-1", "custom-1", "backlog", true);
    const user = userEvent.setup();
    await user.click(await screen.findByRole("button", { name: "Edit YAML" }));
    const invalid = savedYaml.replace("use: task_enters_stage", "use: pull_request_updated");
    fireEvent.change(screen.getByLabelText("Automation YAML"), { target: { value: invalid } });
    await user.click(screen.getByRole("button", { name: "Open visual editor" }));
    expect(screen.getByRole("alert")).toHaveTextContent("Verify");
    expect(screen.getByLabelText("Automation YAML")).toHaveValue(invalid);
    expect(screen.queryByLabelText("Automation name")).not.toBeInTheDocument();
    fireEvent.change(screen.getByLabelText("Automation YAML"), { target: { value: savedYaml + "# Fixed\n" } });
    await user.click(screen.getByRole("button", { name: "Open visual editor" }));
    expect(screen.getByLabelText("Automation name")).toHaveValue("Preview");
    await user.click(screen.getByRole("button", { name: "Cancel" }));
    const confirmation = within(screen.getByRole("alertdialog"));
    expect(onClose).not.toHaveBeenCalled();
    await user.click(confirmation.getByRole("button", { name: "Discard changes" }));
    expect(onClose).toHaveBeenCalledTimes(1);
    expect(fetch.mock.calls.some(([, init]) => init?.method === "PUT")).toBe(false);
  });

  it("saves the column run setting from the visual editor in the YAML definition", async () => {
    const fetch = vi.spyOn(globalThis, "fetch").mockImplementation(async () => Response.json(savedAutomation));
    const onClose = renderModal("?customAutomation=custom-1", "custom-1", "backlog", true);
    const user = userEvent.setup();

    const runs = within(await screen.findByRole("tablist", { name: "Runs" }));
    expect(runs.getAllByRole("tab")).toHaveLength(2);
    expect(runs.getByRole("tab", { name: "on a task", selected: true })).toBeVisible();
    await user.click(runs.getByRole("tab", { name: "with no task" }));
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1));
    const saved = JSON.parse(String(fetch.mock.calls.at(-1)?.[1]?.body)).yaml;
    expect(saved).toContain("runs_on_task: false");
    expect(saved).toContain("use: manual");
    expect(saved).not.toContain("use: task_enters_stage");
  });

  it("retains a rejected draft and shows the server diagnostic", async () => {
    const fetch = mockAutomationSourceFetch((method) =>
      method === "PUT"
        ? Response.json(
            {
              diagnostics: [
                {
                  path: "steps[0].secrets[0]",
                  line: 10,
                  column: 15,
                  message: "Create this secret in the organization.",
                },
              ],
            },
            { status: 400 },
          )
        : Response.json(savedAutomation),
    );
    const onClose = renderModal("?customAutomation=custom-1", "custom-1", "backlog", true);
    const user = userEvent.setup();
    await user.click(await screen.findByRole("button", { name: "Edit YAML" }));
    const draft = savedYaml.replace("echo original", "echo original\n    secrets: [missing-secret]");
    fireEvent.change(screen.getByLabelText("Automation YAML"), { target: { value: draft } });
    await user.click(screen.getByRole("button", { name: "Save" }));
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "steps[0].secrets[0] · Line 10, column 15 · Create this secret in the organization.",
    );
    expect(screen.getByLabelText("Automation YAML")).toHaveValue(draft);
    expect(onClose).not.toHaveBeenCalled();
    expect(fetch.mock.calls.filter(([, init]) => init?.method === "PUT")).toHaveLength(1);
  });

  it("closes cleanly after visiting YAML and Runs without edits", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(Response.json(savedAutomation));
    const onClose = renderModal("?customAutomation=custom-1", "custom-1", "backlog", true);
    const user = userEvent.setup();
    await user.click(await screen.findByRole("button", { name: "Edit YAML" }));
    await user.click(screen.getByRole("tab", { name: "Runs" }));
    expect(screen.queryByRole("button", { name: "Open visual editor" })).not.toBeInTheDocument();
    await user.click(screen.getByRole("tab", { name: "Configuration" }));
    expect(screen.getByLabelText("Automation YAML")).toHaveValue(savedYaml);
    await user.click(screen.getByRole("button", { name: "Cancel" }));
    expect(onClose).toHaveBeenCalledTimes(1);
    expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument();
  });

  it("preserves a dirty draft when the browser Back confirmation is declined", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(Response.json(savedAutomation));
    const confirm = vi.spyOn(window, "confirm").mockReturnValueOnce(false).mockReturnValueOnce(true);
    const boardPath = "/org/workspaces/sp/lines/line";
    window.history.replaceState(null, "", boardPath);

    function Board() {
      const { search } = useLocation();
      return search ? (
        <CustomAutomationSetupModal lineId="line" automationId="custom-1" yamlEnabled onClose={() => {}} />
      ) : (
        <Link to="?customAutomation=custom-1">Open automation</Link>
      );
    }

    render(
      <BrowserRouter>
        <AutomationModalProviders>
          <Board />
        </AutomationModalProviders>
      </BrowserRouter>,
    );
    const user = userEvent.setup();
    await user.click(screen.getByRole("link", { name: "Open automation" }));
    fireEvent.change(await screen.findByLabelText("Automation name"), { target: { value: "Unsaved preview" } });
    await user.click(screen.getByRole("tab", { name: "Runs" }));
    await act(async () => window.history.back());
    expect(await screen.findByLabelText("Automation name")).toHaveValue("Unsaved preview");
    expect(confirm).not.toHaveBeenCalled();

    await act(async () => window.history.back());
    await waitFor(() => expect(confirm).toHaveBeenCalledTimes(1));
    await waitFor(() => expect(window.location.search).toBe("?customAutomation=custom-1"));
    expect(screen.getByLabelText("Automation name")).toHaveValue("Unsaved preview");

    await act(async () => window.history.back());
    expect(await screen.findByRole("link", { name: "Open automation" })).toBeVisible();
    expect(confirm).toHaveBeenCalledTimes(2);
  });
});
