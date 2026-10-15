import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "bun:test";

import { FirstRunSignedInAsLine } from "./FirstRunSignedInAsLine";

const testIds = {
  signedInAs: "signed-in",
  switchAccount: "switch-account",
};

describe("FirstRunSignedInAsLine", () => {
  it("keeps the account name when that name also appears in the sentence", () => {
    render(
      <FirstRunSignedInAsLine
        login="as"
        activeKey="account-1"
        identities={[{ key: "account-1", login: "as" }]}
        disabled={false}
        signedInPrefix="You are signed in to Bitbucket as "
        signedInSuffix="."
        connectAnotherLabel="Connect another Bitbucket account"
        testIds={testIds}
      />,
    );

    const signedIn = screen.getByTestId("signed-in");
    expect(signedIn).toHaveTextContent("You are signed in to Bitbucket as as.");
    expect(signedIn.querySelector("span")).toHaveTextContent("as");
  });
});
