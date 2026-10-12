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

function ControlledRenderer() {
  const [value, setValue] = useState<string>();
  return (
    <>
      <span data-testid="current-value">{value ?? ""}</span>
      <RunnerFleetFieldRenderer
        field={field}
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
  });

  it("stores the selected fleet id", async () => {
    render(<ControlledRenderer />);

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
    expect(screen.getByText("Loading machine types...")).toBeInTheDocument();

    mockUseOrganizationRunnerFleets.mockReturnValueOnce({
      isLoading: false,
      isError: true,
    } as unknown as ReturnType<typeof useOrganizationRunnerFleets>);
    rerender(<RunnerFleetFieldRenderer field={field} value="" onChange={vi.fn()} organizationId="org-1" />);
    expect(screen.getByText("SuperPlane could not load machine types. Try again.")).toBeInTheDocument();

    mockUseOrganizationRunnerFleets.mockReturnValueOnce({
      data: { fleets: [] },
      isLoading: false,
      isError: false,
    } as unknown as ReturnType<typeof useOrganizationRunnerFleets>);
    rerender(<RunnerFleetFieldRenderer field={field} value="" onChange={vi.fn()} organizationId="org-1" />);
    expect(screen.getByText("No machine types are available.")).toBeInTheDocument();
  });
});
