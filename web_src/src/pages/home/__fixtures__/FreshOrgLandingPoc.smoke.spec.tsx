import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeAll, describe, expect, it } from "vitest";

import { client } from "@/api-client/client.gen";

import { HomePageHarness } from "./HomePageHarness";
import { emptyHomePageFixture } from "./homePageResponses";

describe("FreshOrgLanding story smoke", () => {
  beforeAll(() => {
    client.setConfig({ baseUrl: "http://localhost" });
  });

  it("renders the Software Factory hero and secondary actions", async () => {
    const user = userEvent.setup();
    render(<HomePageHarness fixture={emptyHomePageFixture} pathSuffix="apps/welcome" />);

    expect(
      await screen.findByRole("heading", { name: "Set up your Software Factory" }, { timeout: 5000 }),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /get started/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /create a blank app/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /browse starter apps/i })).toBeInTheDocument();
    expect(screen.queryByText("Starter apps")).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /browse starter apps/i }));
    expect(screen.getByText("Starter apps")).toBeInTheDocument();
  });
});
