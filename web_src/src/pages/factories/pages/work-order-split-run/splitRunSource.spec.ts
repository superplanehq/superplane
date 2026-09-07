import { describe, expect, it } from "vitest";

import { PRIMARY_FACTORY_ID, STORYBOOK_ME_USER_ID } from "../../__fixtures__/factoryPageIds";
import {
  DRAFT_WORK_ORDER,
  OPEN_WORK_ORDER_SECONDARY,
  QUESTION_WORK_ORDER,
  RUNNING_WORK_ORDER,
  SENTRY_DRAFT_WORK_ORDER,
  SLACK_DRAFT_WORK_ORDER,
} from "../../__fixtures__/factoryPageResponses";
import { lineMetricsFactoriesFixture } from "../../__fixtures__/lineMetricsFactoriesFixture";
import { REVIEW_CANDIDATE_WORK_ORDERS } from "../onboarding/first-run/reviewCandidates";
import { collectSplitRunArtifacts, splitRunLinkedArtifacts } from "./splitRunPopupModel";
import { splitRunFixtureForWorkOrder } from "./splitRunMocks";
import { sourceTicketLabel, splitRunSourceForOrder } from "./splitRunSource";

describe("sourceTicketLabel", () => {
  it("uses owner/repo#number for GitHub issues", () => {
    expect(sourceTicketLabel("https://github.com/acme/payments-service/issues/842")).toBe("acme/payments-service#842");
  });

  it("uses org#id for Sentry, PagerDuty, and Slack hosts", () => {
    expect(sourceTicketLabel("https://superplane.sentry.io/issues/7670162495/events/abc/")).toBe(
      "superplane#7670162495",
    );
    expect(sourceTicketLabel("https://acme.pagerduty.com/incidents/P123ABC")).toBe("acme#P123ABC");
    expect(sourceTicketLabel("https://acme.slack.com/archives/C0REFUNDS/p1710000000000000")).toBe("acme#C0REFUNDS");
  });

  it("uses the task id alone for Productive.io, whose link holds an organization id", () => {
    expect(sourceTicketLabel("https://app.productive.io/48521/tasks/19976991")).toBe("#19976991");
  });
});

describe("splitRunSourceForOrder", () => {
  it("uses the GitHub issue for a review candidate even when a person created the draft", () => {
    const source = splitRunSourceForOrder(REVIEW_CANDIDATE_WORK_ORDERS[0]!);
    expect(source).toEqual(
      expect.objectContaining({
        kind: "intake",
        name: "GitHub issues",
        ticket: {
          label: "acme/payments-service#842",
          href: "https://github.com/acme/payments-service/issues/842",
        },
      }),
    );
  });

  it("uses the Sentry issue for the HTTP 500 review candidate", () => {
    const pay844 = REVIEW_CANDIDATE_WORK_ORDERS.find((order) => order.key === "PAY-844");
    const source = splitRunSourceForOrder(pay844!);
    expect(source).toEqual(
      expect.objectContaining({
        kind: "intake",
        name: "Sentry exceptions",
        ticket: {
          label: "superplane#7670162495",
          href: "https://superplane.sentry.io/issues/7670162495/",
        },
      }),
    );
  });

  it("uses the persisted origin label when the order has one", () => {
    expect(
      splitRunSourceForOrder({
        ...DRAFT_WORK_ORDER,
        origin: {
          url: "https://github.com/acme/payments/issues/12",
          label: "acme/payments#12",
        },
      }),
    ).toEqual(
      expect.objectContaining({
        kind: "intake",
        name: "GitHub issues",
        ticket: { label: "acme/payments#12", href: "https://github.com/acme/payments/issues/12" },
      }),
    );
  });

  it("maps automation-created orders without origin to the intake source", () => {
    const sentry = splitRunSourceForOrder({ ...OPEN_WORK_ORDER_SECONDARY, origin: undefined });
    expect(sentry).toEqual(expect.objectContaining({ kind: "intake", name: "Sentry exceptions" }));
    expect(sentry).not.toHaveProperty("ticket");

    const slack = splitRunSourceForOrder({ ...QUESTION_WORK_ORDER, origin: undefined });
    expect(slack).toEqual(expect.objectContaining({ kind: "intake", name: "Slack" }));
    expect(slack).not.toHaveProperty("ticket");
  });

  it("maps ingest, Sentry, and Slack automations to the drawer intake names", () => {
    expect(splitRunSourceForOrder(RUNNING_WORK_ORDER)).toEqual(
      expect.objectContaining({
        kind: "intake",
        name: "GitHub issues",
        ticket: { label: "acme/payments-service#103", href: expect.stringContaining("/issues/103") },
      }),
    );
    expect(splitRunSourceForOrder(SENTRY_DRAFT_WORK_ORDER)).toEqual(
      expect.objectContaining({
        kind: "intake",
        name: "Sentry exceptions",
        ticket: { label: expect.stringMatching(/^superplane#/), href: expect.stringContaining("sentry.io") },
      }),
    );
    expect(splitRunSourceForOrder(SLACK_DRAFT_WORK_ORDER)).toEqual(
      expect.objectContaining({
        kind: "intake",
        name: "Slack",
        ticket: { label: expect.stringMatching(/^acme#/), href: expect.stringContaining("slack.com") },
      }),
    );
  });

  // A Productive.io webhook carries no link to the task, so the order has no
  // origin and the automation is the only evidence of where it came from.
  // Naming the source after the automation keeps it from reading as GitHub.
  it("names the Productive.io intake when the order has no origin", () => {
    expect(
      splitRunSourceForOrder({
        ...DRAFT_WORK_ORDER,
        origin: undefined,
        createdBy: { automation: { appId: "app-productive-intake", appName: "Productive.io tasks" } },
      }),
    ).toEqual(expect.objectContaining({ kind: "intake", name: "Productive.io tasks", iconAlt: "Productive.io" }));
  });

  it("names the Productive.io intake from a task link", () => {
    expect(
      splitRunSourceForOrder({
        ...DRAFT_WORK_ORDER,
        origin: { url: "https://app.productive.io/1-acme/tasks/task/19976991" },
      }),
    ).toEqual(expect.objectContaining({ kind: "intake", name: "Productive.io tasks" }));
  });

  it("uses the person and Created manually when a person opened the task", () => {
    expect(splitRunSourceForOrder(DRAFT_WORK_ORDER)).toEqual(
      expect.objectContaining({
        kind: "manual",
        detail: "Created manually",
        person: expect.objectContaining({ name: "Leonardo DiCaprio" }),
      }),
    );
  });

  it("resolves the source person's avatar from the org members list when one is available", () => {
    const resolveUser = (userId: string | undefined, name?: string) =>
      userId
        ? { id: userId, name: name ?? "Member", initials: "M", avatarUrl: "https://example.com/avatar.jpg" }
        : null;

    const source = splitRunSourceForOrder(DRAFT_WORK_ORDER, resolveUser);
    expect(source).toEqual(
      expect.objectContaining({
        kind: "manual",
        person: expect.objectContaining({
          id: STORYBOOK_ME_USER_ID,
          name: "Leonardo DiCaprio",
          avatarUrl: "https://example.com/avatar.jpg",
        }),
      }),
    );
  });

  it("falls back to initials when the org member has no avatar image", () => {
    const resolveUser = (userId: string | undefined, name?: string) =>
      userId ? { id: userId, name: name ?? "Member", initials: "M" } : null;

    const source = splitRunSourceForOrder(DRAFT_WORK_ORDER, resolveUser);
    expect(source).toEqual(
      expect.objectContaining({
        kind: "manual",
        person: expect.objectContaining({ id: STORYBOOK_ME_USER_ID, name: "Leonardo DiCaprio" }),
      }),
    );
    if (source.kind === "manual") {
      expect(source.person.avatarUrl).toBeUndefined();
    }
  });

  it("fills Source for every task on the populated line board", () => {
    const orders = lineMetricsFactoriesFixture.workOrdersByFactoryId[PRIMARY_FACTORY_ID] ?? [];
    expect(orders.length).toBeGreaterThan(0);
    for (const order of orders) {
      const source = splitRunSourceForOrder(order);
      if (source.kind === "intake") {
        expect(source.name, order.id).toBeTruthy();
        if (source.ticket) {
          expect(source.ticket.href, order.id).toMatch(/^https?:/);
          expect(source.ticket.label, order.id).toBeTruthy();
        }
      } else {
        expect(source.person.name, order.id).toBeTruthy();
        expect(source.detail, order.id).toBe("Created manually");
      }
    }
  });
});

describe("splitRunLinkedArtifacts", () => {
  it("keeps plan.md and drops the origin ticket from Artifacts", () => {
    const fixture = splitRunFixtureForWorkOrder(REVIEW_CANDIDATE_WORK_ORDERS[0]);
    const artifacts = collectSplitRunArtifacts(fixture);
    const linked = splitRunLinkedArtifacts(artifacts, fixture.source);

    expect(linked.some((artifact) => artifact.id?.endsWith("-plan"))).toBe(true);
    expect(linked.some((artifact) => artifact.id?.endsWith("-issue-link"))).toBe(false);
    expect(linked.some((artifact) => artifact.id?.endsWith("-details"))).toBe(false);
  });
});
