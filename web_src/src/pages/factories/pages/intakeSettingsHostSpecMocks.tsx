import type * as CanvasDataModule from "@/hooks/useCanvasData";
import type * as ComponentDataModule from "@/hooks/useComponentData";
import type * as FactoryIntakeDataModule from "@/hooks/useFactoryIntakeData";
import type * as IntegrationsModule from "@/hooks/useIntegrations";
import type * as OrganizationDataModule from "@/hooks/useOrganizationData";
import { unmockedSrc } from "@/test/unmockedModule";
import { vi } from "bun:test";

export const {
  useCanvas,
  useTriggers,
  useComponents,
  useAvailableIntegrations,
  useIntegrationResources,
  updateIntake,
  deleteIntake,
  useInfiniteCanvasRuns,
  useDescribeRun,
  useEventExecutions,
} = vi.hoisted(() => ({
  useCanvas: vi.fn(),
  useTriggers: vi.fn(),
  useComponents: vi.fn(),
  useAvailableIntegrations: vi.fn(),
  useIntegrationResources: vi.fn(),
  updateIntake: vi.fn(),
  deleteIntake: vi.fn(),
  useInfiniteCanvasRuns: vi.fn(),
  useDescribeRun: vi.fn(),
  useEventExecutions: vi.fn(),
}));

vi.mock("@monaco-editor/react", () => {
  function MockMonacoEditor({ value, onChange }: { value?: string; onChange?: (value: string | undefined) => void }) {
    return <textarea value={value ?? ""} onChange={(event) => onChange?.(event.target.value)} />;
  }
  return { default: MockMonacoEditor, Editor: MockMonacoEditor };
});

vi.mock("@/hooks/useCanvasData", () => ({
  ...unmockedSrc<typeof CanvasDataModule>("hooks/useCanvasData"),
  useCanvas,
  useTriggers,
  useInfiniteCanvasRuns,
  useDescribeRun,
  useEventExecutions,
}));

vi.mock("@/hooks/useComponentData", () => ({
  ...unmockedSrc<typeof ComponentDataModule>("hooks/useComponentData"),
  useComponents,
}));

vi.mock("@/hooks/useIntegrations", () => ({
  ...unmockedSrc<typeof IntegrationsModule>("hooks/useIntegrations"),
  useAvailableIntegrations,
  useIntegrationResources,
}));

vi.mock("@/hooks/useFactoryIntakeData", () => ({
  ...unmockedSrc<typeof FactoryIntakeDataModule>("hooks/useFactoryIntakeData"),
  useUpdateFactoryIntake: () => ({ mutateAsync: updateIntake, isPending: false, error: null }),
  useDeleteFactoryIntake: () => ({ mutateAsync: deleteIntake, isPending: false, error: null }),
}));

vi.mock("@/hooks/useOrganizationData", () => ({
  ...unmockedSrc<typeof OrganizationDataModule>("hooks/useOrganizationData"),
  useOrganizationUsers: () => ({
    data: [{ metadata: { id: "alice" }, spec: { displayName: "Alice" } }],
    isLoading: false,
    isError: false,
  }),
}));

vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({ currentUserId: "alice", canAct: () => false }),
}));
