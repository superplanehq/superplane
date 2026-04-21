import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { ComponentBase } from "./index";

describe("ComponentBase event title resolution", () => {
  it("uses defaultEventTitle when an event section has no explicit title", () => {
    render(
      <ComponentBase
        title="Component"
        iconSlug="bolt"
        defaultEventTitle="Root Event Title"
        eventSections={[
          {
            eventId: "event-1",
            eventState: "success",
            eventSubtitle: "just now",
            receivedAt: new Date(),
          },
        ]}
      />,
    );

    expect(screen.getByText("Root Event Title")).toBeInTheDocument();
  });

  it("prefers the explicit section title over defaultEventTitle", () => {
    render(
      <ComponentBase
        title="Component"
        iconSlug="bolt"
        defaultEventTitle="Root Event Title"
        eventSections={[
          {
            eventId: "event-1",
            eventTitle: "Explicit Event Title",
            eventState: "success",
            eventSubtitle: "just now",
            receivedAt: new Date(),
          },
        ]}
      />,
    );

    expect(screen.getByText("Explicit Event Title")).toBeInTheDocument();
    expect(screen.queryByText("Root Event Title")).not.toBeInTheDocument();
  });
});
