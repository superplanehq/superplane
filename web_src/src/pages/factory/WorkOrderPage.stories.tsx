import type { Meta, StoryObj } from "@storybook/react-vite";
import { fn } from "storybook/test";

import { workOrderEvents, workOrders } from "./__fixtures__/factoryFixtures";
import { WorkOrderPage } from "./WorkOrderPage";

/**
 * Dedicated page for one Work Order, designed from
 * `docs/prd/software-factory.md`.
 *
 * Two parts, in the PRD's order: the work description, which stays visible as
 * the context for the implementation, then the chronology — "the main
 * structural model of the page". Oldest event at the top, newest appended at
 * the bottom, with conversations, decisions, approvals and steering in the same
 * durable rail as Automation events.
 */
const meta = {
  title: "Pages/Work Order",
  component: WorkOrderPage,
  parameters: {
    layout: "fullscreen",
  },
} satisfies Meta<typeof WorkOrderPage>;

export default meta;

type Story = StoryObj<typeof meta>;

const factory = { id: "factory-1", name: "Payments Factory" };

const handlers = {
  onBackToFactory: fn(),
  onApprove: fn(),
  onSteer: fn(),
  onRetry: fn(),
};

/**
 * A draft awaiting approval — nothing has run yet, so the chronology is two
 * events long and the only action is Approve.
 */
export const Draft: Story = {
  args: {
    ...handlers,
    data: {
      factory,
      workOrder: workOrders[1],
      events: workOrderEvents.slice(0, 1),
    },
  },
};

/**
 * Blocked on a person: the Automation asked a question and stopped. The
 * approval request carries the accent, because it is the one thing on the page
 * that needs a human.
 */
export const NeedsAttention: Story = {
  args: {
    ...handlers,
    data: {
      factory,
      workOrder: workOrders[0],
      events: workOrderEvents
        .slice(0, 5)
        .map((event, index) => (index === 4 ? { ...event, awaitingResponse: true } : event)),
    },
  },
};

/** Mid-flight after the question was answered and the plan narrowed. */
export const Running: Story = {
  args: {
    ...handlers,
    data: {
      factory,
      workOrder: { ...workOrders[2], attention: undefined },
      events: workOrderEvents.slice(0, 9),
    },
  },
};

/**
 * The interesting case for an append-only record: the Work Order succeeded,
 * was reopened after review, and the second attempt is appended below the first
 * rather than replacing it. "New attempt" marks the seam.
 */
export const ReopenedAfterSuccess: Story = {
  args: {
    ...handlers,
    data: {
      factory,
      workOrder: {
        ...workOrders[0],
        state: "ready",
        attention: undefined,
        activity: "Adding a table test for the duplicate path",
        pullRequests: [
          {
            provider: "github",
            repository: "acme/payments-api",
            number: 1841,
            url: "https://github.com/acme/payments-api/pull/1841",
          },
        ],
      },
      events: workOrderEvents,
    },
  },
};

/** A finished Work Order — the composer gives way to Retry, which appends. */
export const Successful: Story = {
  args: {
    ...handlers,
    data: {
      factory,
      workOrder: {
        ...workOrders[4],
        pullRequests: [
          {
            provider: "github",
            repository: "acme/payments-api",
            number: 1838,
            url: "https://github.com/acme/payments-api/pull/1838",
          },
        ],
      },
      events: workOrderEvents.slice(0, 10),
    },
  },
};
