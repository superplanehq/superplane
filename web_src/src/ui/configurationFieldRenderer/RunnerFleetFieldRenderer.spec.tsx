import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import type { ConfigurationField } from "@/api-client";
import { useOrganizationRunnerFleets } from "@/hooks/useOrganizationRunnerFleets";
import { RunnerFleetFieldRenderer } from "./RunnerFleetFieldRenderer";

vi.mock("@/hooks/useOrganizationRunnerFleets", () => ({
  useOrganizationRunnerFleets: vi.fn(),
}));

const mockUseOrganizationRunnerFleets = vi.mocked(useOrganizationRunnerFleets);
const field: ConfigurationField = {
  name: "machineType",
  label: "Machine type",
  type: "runner-fleet",
  required: true,
};

function ControlledRenderer({ fieldName = "machineType" }: { fieldName?: string }) {
  const [value, setValue] = useState<string>();
  return (
    <>
      <span data-testid="current-value">{value ?? ""}</span>
      <RunnerFleetFieldRenderer
        field={{ ...field, name: fieldName }}
        value={value}
        onChange={(nextValue) => setValue(nextValue as string | undefined)}
        organizationId="org-1"
      />
    </>
  );
}

beforeEach(() => {
  mockUseOrganizationRunnerFleets.mockReturnValue({
    data: {
      fleets: [
        {
          id: "e1-large-amd64",
          scope: "installation",
          spec: { operatingSystem: "linux", architecture: "amd64", cpuMillicores: 8000, memoryMb: 32768, diskGb: 30 },
        },
        {
          id: "aws-large-amd64",
          scope: "organization",
          spec: { operatingSystem: "linux", architecture: "amd64", cpuMillicores: 16000, memoryMb: 65536, diskGb: 30 },
        },
      ],
    },
    isLoading: false,
    isError: false,
  } as unknown as ReturnType<typeof useOrganizationRunnerFleets>);
});

describe("RunnerFleetFieldRenderer", () => {
  it("lists the machine types available to the organization", async () => {
    render(<ControlledRenderer />);

    await userEvent.click(screen.getByRole("combobox"));
    expect(screen.getByText("e1-large-amd64 (linux, 8 vCPU, 32GB)")).toBeInTheDocument();
    expect(screen.getByText("aws-large-amd64 (linux, 16 vCPU, 64GB)")).toBeInTheDocument();
    expect(screen.getByRole("group", { name: "Organization fleets" })).toContainElement(
      screen.getByRole("option", { name: "aws-large-amd64 (linux, 16 vCPU, 64GB)" }),
    );
    expect(screen.getByRole("group", { name: "Installation fleets" })).toContainElement(
      screen.getByRole("option", { name: "e1-large-amd64 (linux, 8 vCPU, 32GB)" }),
    );
  });

  it.each(["machineType", "machine_type"])("stores the selected fleet id for %s", async (fieldName) => {
    render(<ControlledRenderer fieldName={fieldName} />);

    await userEvent.click(screen.getByRole("combobox"));
    await userEvent.click(screen.getByText("aws-large-amd64 (linux, 16 vCPU, 64GB)"));
    expect(screen.getByTestId("current-value")).toHaveTextContent("aws-large-amd64");
  });

  it("shows loading, error, and empty states", () => {
    mockUseOrganizationRunnerFleets.mockReturnValueOnce({
      isLoading: true,
      isError: false,
    } as unknown as ReturnType<typeof useOrganizationRunnerFleets>);
    const { rerender } = render(
      <RunnerFleetFieldRenderer field={field} value="" onChange={vi.fn()} organizationId="org-1" />,
    );
    expect(screen.getByText("Loading runner fleets...")).toBeInTheDocument();

    mockUseOrganizationRunnerFleets.mockReturnValueOnce({
      isLoading: false,
      isError: true,
    } as unknown as ReturnType<typeof useOrganizationRunnerFleets>);
    rerender(<RunnerFleetFieldRenderer field={field} value="" onChange={vi.fn()} organizationId="org-1" />);
    expect(screen.getByText("SuperPlane could not load runner fleets. Try again.")).toBeInTheDocument();

    mockUseOrganizationRunnerFleets.mockReturnValueOnce({
      data: { fleets: [] },
      isLoading: false,
      isError: false,
    } as unknown as ReturnType<typeof useOrganizationRunnerFleets>);
    rerender(<RunnerFleetFieldRenderer field={field} value="" onChange={vi.fn()} organizationId="org-1" />);
    expect(screen.getByText("No runner fleets are available.")).toBeInTheDocument();
  });

  it("retains a saved fleet across availability refreshes without changing configuration", () => {
    const onChange = vi.fn();
    const props = { field, value: "aws-large-amd64", onChange, organizationId: "org-1" };
    const { rerender } = render(<RunnerFleetFieldRenderer {...props} />);
    expect(screen.getByRole("combobox")).toHaveTextContent("aws-large-amd64");
    mockUseOrganizationRunnerFleets.mockReturnValueOnce({
      data: { fleets: [] },
      isLoading: false,
      isError: false,
    } as unknown as ReturnType<typeof useOrganizationRunnerFleets>);
    rerender(<RunnerFleetFieldRenderer {...props} />);
    expect(screen.getByRole("combobox")).toHaveTextContent("aws-large-amd64 (Unavailable)");
    expect(screen.getByText("Select an available runner fleet before new work can execute.")).toBeInTheDocument();
    rerender(<RunnerFleetFieldRenderer {...props} />);
    expect(screen.getByRole("combobox")).not.toHaveTextContent("Unavailable");
    expect(onChange).not.toHaveBeenCalled();
  });

  it("disables selection for read-only users", async () => {
    const onChange = vi.fn();
    render(
      <RunnerFleetFieldRenderer
        field={field}
        value="aws-large-amd64"
        onChange={onChange}
        organizationId="org-1"
        readOnly
      />,
    );
    expect(screen.getByRole("combobox")).toBeDisabled();
    await userEvent.click(screen.getByRole("combobox"));
    expect(screen.queryByRole("option")).not.toBeInTheDocument();
    expect(onChange).not.toHaveBeenCalled();
  });

  it.each([
    { isLoading: true, isError: false },
    { isLoading: false, isError: true },
  ])("keeps a saved fleet visible while availability is unresolved: %j", (state) => {
    const onChange = vi.fn();
    mockUseOrganizationRunnerFleets.mockReturnValueOnce(state as ReturnType<typeof useOrganizationRunnerFleets>);
    render(<RunnerFleetFieldRenderer field={field} value="saved-fleet" onChange={onChange} organizationId="org-1" />);
    expect(screen.getByRole("combobox")).toHaveTextContent("saved-fleet");
    expect(screen.getByRole("combobox")).not.toHaveTextContent("Unavailable");
    expect(onChange).not.toHaveBeenCalled();
  });

  it("does not classify expressions as unavailable fleet identifiers", () => {
    const onChange = vi.fn();
    render(
      <RunnerFleetFieldRenderer
        field={{ ...field, name: "machine_type" }}
        value="{{ $.fleet }}"
        onChange={onChange}
        organizationId="org-1"
      />,
    );
    expect(screen.getByRole("combobox")).toHaveTextContent("{{ $.fleet }}");
    expect(screen.getByRole("combobox")).not.toHaveTextContent("Unavailable");
    expect(onChange).not.toHaveBeenCalled();
  });
});
