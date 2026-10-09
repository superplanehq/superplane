import React, { useCallback, useEffect, useState } from "react";
import ForgeReconciler, { Button, Heading, SectionMessage, Text } from "@forge/react";
import { invokeRemote } from "@forge/bridge";

const BOOTSTRAP_PATH = "/api/v1/bitbucket/forge/bootstrap";

// Setup page under Apps → SuperPlane. Opening this page confirms the
// installation: the call below travels through Forge with the system token
// enabled, so SuperPlane receives the same verified delivery as the
// lifecycle event. Show confirmation only after SuperPlane accepts it.
function Setup() {
  const [status, setStatus] = useState("confirming");

  const confirm = useCallback(async () => {
    setStatus("confirming");
    try {
      await invokeRemote({
        path: BOOTSTRAP_PATH,
        method: "POST",
      });
      setStatus("confirmed");
    } catch {
      setStatus("failed");
    }
  }, []);

  useEffect(() => {
    void confirm();
  }, [confirm]);

  if (status === "confirmed") {
    return (
      <>
        <Heading as="h1">SuperPlane</Heading>
        <SectionMessage title="Installation confirmed" appearance="confirmation">
          <Text>SuperPlane received this workspace. Return to the SuperPlane tab to continue setup.</Text>
        </SectionMessage>
      </>
    );
  }

  if (status === "failed") {
    return (
      <>
        <Heading as="h1">SuperPlane</Heading>
        <SectionMessage title="Confirmation failed" appearance="error">
          <Text>SuperPlane did not receive this workspace. Try again.</Text>
        </SectionMessage>
        <Button appearance="primary" onClick={() => void confirm()}>
          Try again
        </Button>
      </>
    );
  }

  return (
    <>
      <Heading as="h1">SuperPlane</Heading>
      <Text>Confirming this workspace with SuperPlane…</Text>
    </>
  );
}

ForgeReconciler.render(
  <React.StrictMode>
    <Setup />
  </React.StrictMode>,
);
