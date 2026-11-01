import { useEffect, useState, type ReactNode } from "react";
import type { OrganizationsCreateFleetRunnerResponse, SuperplaneOrganizationsFleet } from "@/api-client";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { CopyButton } from "@/ui/CopyButton";
import { defaultRunnerImage, runnerCommand, runnerDownloadCommand } from "@/lib/runnerSetup";

export function RunnerInstructions({
  fleet,
  registration,
  children,
}: {
  fleet: SuperplaneOrganizationsFleet;
  registration: OrganizationsCreateFleetRunnerResponse | null;
  children: (ready: boolean) => ReactNode;
}) {
  const [method, setMethod] = useState("binary");
  const [image, setImage] = useState(() => defaultRunnerImage(fleet));
  const download = runnerDownloadCommand(fleet);
  return (
    <Tabs value={method} onValueChange={setMethod}>
      <TabsList aria-label="Setup method">
        <TabsTrigger value="binary">Binary</TabsTrigger>
        <TabsTrigger value="docker">Docker</TabsTrigger>
      </TabsList>
      <TabsContent value="binary" className="space-y-4 pt-4">
        <p className="text-muted-foreground">
          Download the runner for {fleet.spec?.operatingSystem || "linux"} / {fleet.spec?.architecture || "amd64"}, then
          run it on your machine.
        </p>
        {download ? (
          <CommandBlock command={download} label="1. Download and extract" copyLabel="Copy download command" />
        ) : (
          <p className="rounded-md bg-muted/50 p-3 text-muted-foreground">
            Use a runner binary built for this fleet's version ({fleet.runnerVersion}) and platform. Published GitHub
            binaries support Linux amd64 and arm64.
          </p>
        )}
      </TabsContent>
      <TabsContent value="docker" className="space-y-4 pt-4">
        <p className="text-muted-foreground">
          Run the runner in a Docker container. Use an image that matches this fleet's runner version (
          {fleet.runnerVersion}).
        </p>
        <div>
          <Label htmlFor="runner-image" className="mb-1.5 block text-[12px]">
            Runner image
          </Label>
          <Input
            id="runner-image"
            value={image}
            onChange={(event) => setImage(event.target.value)}
            placeholder="Registry and image tag"
            className="text-[13px]"
          />
        </div>
        <p className="text-[12px] text-muted-foreground">
          The container must be able to reach SuperPlane. Localhost URLs use host.docker.internal in the command.
        </p>
      </TabsContent>
      <div className="space-y-4 pt-4">
        {children(method === "binary" || !!image.trim())}
        {registration && (method === "binary" || image.trim()) && (
          <RegistrationCommand
            key={registration.registrationToken}
            registration={registration}
            method={method}
            image={image.trim()}
            fleet={fleet}
          />
        )}
        {registration && method === "docker" && !image.trim() && (
          <p className="text-muted-foreground">Enter a runner image to show the Docker command.</p>
        )}
      </div>
    </Tabs>
  );
}

function RegistrationCommand({
  registration,
  method,
  image,
  fleet,
}: {
  registration: OrganizationsCreateFleetRunnerResponse;
  method: string;
  image: string;
  fleet: SuperplaneOrganizationsFleet;
}) {
  const expiresAt = Date.parse(registration.registrationExpiresAt!);
  const [expired, setExpired] = useState(Date.now() >= expiresAt);
  useEffect(() => {
    if (expired) return;
    const timer = setTimeout(() => setExpired(true), Math.max(0, expiresAt - Date.now()));
    return () => clearTimeout(timer);
  }, [expiresAt, expired]);
  if (expired) return <p>Checking final registration state before expiration.</p>;
  return (
    <div className="space-y-3">
      <CommandBlock
        command={runnerCommand(registration, method, image, fleet)}
        label="Run the runner"
        copyLabel="Copy command"
      />
      <p role="status" className="text-muted-foreground">
        Waiting for runner registration.
      </p>
      <p className="text-[12px] text-muted-foreground">Registration expires: {new Date(expiresAt).toLocaleString()}</p>
    </div>
  );
}

function CommandBlock({ command, label, copyLabel }: { command: string; label: string; copyLabel: string }) {
  const [copyError, setCopyError] = useState("");
  return (
    <div
      className="overflow-hidden rounded-md border border-border bg-muted/40"
      onClickCapture={() => setCopyError("")}
    >
      <div className="flex items-center justify-between border-b border-border px-3 py-2">
        <h4 className="text-[12px] font-medium">{label}</h4>
        <CopyButton
          text={command}
          ariaLabel={copyLabel}
          onCopyError={() => setCopyError("Could not copy the command. Copy it manually.")}
        />
      </div>
      <pre className="whitespace-pre-wrap break-all p-3 font-mono text-[12px]">
        <code>{command}</code>
      </pre>
      {copyError && (
        <p role="alert" className="px-3 pb-3 text-[12px] text-destructive">
          {copyError}
        </p>
      )}
    </div>
  );
}
