import { render, screen } from "@testing-library/react";
import { beforeAll, describe, expect, it } from "bun:test";

import { client } from "@/api-client/client.gen";
import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import { PRIMARY_FACTORY_KEY } from "../../__fixtures__/factoryPageResponses";
import {
  CONNECTED_SETUP_INTEGRATIONS,
  SETUP_ANSWERS,
  factoriesFixtureWithSetupAnswers,
} from "../../__fixtures__/setupStoryFixtures";

describe("Legacy repository settings routes", () => {
  beforeAll(() => client.setConfig({ baseUrl: "http://localhost" }));

  it.each(["workspace/repository", "repository", "repositories"])(
    "opens Agent from settings/%s by default",
    async (path) => {
      render(
        <FactoriesHarness
          pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/${path}`}
          factoriesFixture={factoriesFixtureWithSetupAnswers(SETUP_ANSWERS.agent)}
          orgIntegrations={CONNECTED_SETUP_INTEGRATIONS}
        />,
      );
      expect(await screen.findByTestId("factory-settings-agent")).toBeInTheDocument();
      expect(screen.queryByTestId("factory-settings-repository-save")).not.toBeInTheDocument();
    },
  );
});
