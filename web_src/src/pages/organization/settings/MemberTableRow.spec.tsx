import { describe, expect, it } from "bun:test";
import { render, screen } from "@testing-library/react";

import { MemberIdentity } from "./MemberTableRow";

describe("MemberIdentity", () => {
  it("shows an SSO badge for a just-in-time member", () => {
    render(<MemberIdentity name="Ada" email="ada@example.com" initials="A" provisionedBy="jit" />);
    expect(screen.getByText("SSO")).toBeInTheDocument();
  });

  it("shows a SCIM badge for a provisioned member", () => {
    render(<MemberIdentity name="Ada" email="ada@example.com" initials="A" provisionedBy="scim" />);
    expect(screen.getByText("SCIM")).toBeInTheDocument();
  });
});
