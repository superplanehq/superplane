import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "bun:test";

import { DATADOG_ISSUE_SEARCH_FAILED_NOTICE_ID, INTEGRATION_LIST_NOTICE_TYPE } from "@/lib/integrationListNotice";

import { DatadogServicePicker } from "./DatadogServicePicker";
import { DATADOG_INTAKE_SETUP_COPY } from "./datadogIntakeSetupCopy";

const services = [
  { id: "checkout", name: "checkout", type: "service" },
  {
    id: DATADOG_ISSUE_SEARCH_FAILED_NOTICE_ID,
    name: "The open-issue search failed. Services found only on open issues are missing.",
    type: INTEGRATION_LIST_NOTICE_TYPE,
  },
];

describe("DatadogServicePicker", () => {
  it("lets the user type a name when the open-issue search failed", async () => {
    const onChange = vi.fn();
    const onRetry = vi.fn();
    const user = userEvent.setup();
    render(
      <DatadogServicePicker
        services={services}
        serviceName=""
        loading={false}
        error={false}
        onChange={onChange}
        onRetry={onRetry}
      />,
    );

    expect(screen.getByText(DATADOG_INTAKE_SETUP_COPY.serviceListIssueSearchFailed)).toBeInTheDocument();
    expect(screen.queryByTestId(`datadog-service-${DATADOG_ISSUE_SEARCH_FAILED_NOTICE_ID}`)).not.toBeInTheDocument();
    expect(screen.getByTestId("datadog-service-checkout")).toBeInTheDocument();

    await user.type(screen.getByTestId("datadog-service-name"), "payments");
    expect(onChange).toHaveBeenCalled();
    await user.click(screen.getByRole("button", { name: DATADOG_INTAKE_SETUP_COPY.wizardRetry }));
    expect(onRetry).toHaveBeenCalled();
  });

  it("does not tell settings users to type a missing service name", () => {
    render(
      <DatadogServicePicker
        services={services}
        serviceName="checkout"
        loading={false}
        error={false}
        onChange={vi.fn()}
        onRetry={vi.fn()}
        showNameField={false}
      />,
    );

    expect(screen.queryByTestId("datadog-service-name")).not.toBeInTheDocument();
    expect(screen.getByText(DATADOG_INTAKE_SETUP_COPY.serviceListIssueSearchFailedNoNameField)).toBeInTheDocument();
    expect(screen.queryByText(DATADOG_INTAKE_SETUP_COPY.serviceListIssueSearchFailed)).not.toBeInTheDocument();
    expect(screen.queryByTestId(`datadog-service-${DATADOG_ISSUE_SEARCH_FAILED_NOTICE_ID}`)).not.toBeInTheDocument();
  });

  it("hides the failure note when the open-issue search succeeds", () => {
    render(
      <DatadogServicePicker
        services={[{ id: "checkout", name: "checkout", type: "service" }]}
        serviceName="checkout"
        loading={false}
        error={false}
        onChange={vi.fn()}
        onRetry={vi.fn()}
        showNameField={false}
      />,
    );

    expect(
      screen.queryByText(DATADOG_INTAKE_SETUP_COPY.serviceListIssueSearchFailedNoNameField),
    ).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: DATADOG_INTAKE_SETUP_COPY.wizardRetry })).not.toBeInTheDocument();
  });
});
