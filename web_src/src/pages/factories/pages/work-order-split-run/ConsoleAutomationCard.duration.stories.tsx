import type { Meta, StoryObj } from "@storybook/react-vite";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useEffect, useState } from "react";

import { Button } from "@/components/ui/button";
import { ComponentStoryShell } from "../../__fixtures__/ComponentStoryShell";
import { withFactoriesTheme } from "../../__fixtures__/factoriesStoryTheme";
import { createConsoleDurationSimulation, type ConsoleDurationSimulation } from "./consoleDurationStoryFixture";
import { ConsoleAutomationCard } from "./redesign/consoleAutomationCard";

const completed = createConsoleDurationSimulation(false);
const live = createConsoleDurationSimulation(true);

const meta = {
  title: "Factories/Pages/Task Split Run/Duration simulation",
  parameters: { layout: "fullscreen", options: { showPanel: false } },
  decorators: [withFactoriesTheme],
} satisfies Meta;

export default meta;
type Story = StoryObj;

function Simulation({ simulation }: { simulation: ConsoleDurationSimulation }) {
  const [replay, setReplay] = useState(0);
  return (
    <ComponentStoryShell className="min-h-svh bg-muted/30 p-6">
      <div className="mx-auto max-w-3xl space-y-5">
        <header className="space-y-2">
          <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">Simulation</p>
          <h1 className="text-xl font-semibold">Task log duration</h1>
          <p className="text-sm text-muted-foreground">Local sample data. No production task or agent runs are used.</p>
          <p className="text-sm">The sample starts with a 28m 20s run and 5m 30s of agent commands.</p>
          <p className="text-sm text-muted-foreground">The agent starts 22m 50s after the run starts.</p>
          <p className="text-sm text-muted-foreground">Expected steps: Checkout 1s, Prompt 29s, and Wait 5m.</p>
          {simulation.initiallyRunning ? (
            <p className="text-sm text-muted-foreground">
              The open Wait step starts at 5m. Finish the run to record its final duration.
            </p>
          ) : null}
        </header>
        <SimulationCard key={replay} simulation={simulation} onReopen={() => setReplay((value) => value + 1)} />
      </div>
    </ComponentStoryShell>
  );
}

function SimulationCard({ simulation, onReopen }: { simulation: ConsoleDurationSimulation; onReopen: () => void }) {
  const [queryClient] = useState(() => new QueryClient({ defaultOptions: { queries: { retry: false } } }));
  const [, setRevision] = useState(0);
  useEffect(() => () => queryClient.clear(), [queryClient]);
  const phase = simulation.phase();
  function finishRun() {
    simulation.finish();
    setRevision((value) => value + 1);
    void queryClient.invalidateQueries();
  }
  return (
    <>
      <div className="flex flex-wrap gap-2">
        <Button variant="outline" onClick={onReopen}>
          Reopen log
        </Button>
        {simulation.initiallyRunning ? (
          <Button disabled={simulation.isFinished()} onClick={finishRun}>
            Finish run
          </Button>
        ) : null}
      </div>
      <QueryClientProvider client={queryClient}>
        <ConsoleAutomationCard
          automation={simulation.automation()}
          phase={phase}
          phases={[phase]}
          organizationId={simulation.organizationId}
          expandIdle
          canStopRun={false}
          actionBusy={false}
        />
      </QueryClientProvider>
    </>
  );
}

export const Completed: Story = {
  name: "Completed commands",
  parameters: { msw: { handlers: completed.handlers } },
  render: () => <Simulation simulation={completed} />,
};

export const LiveWaiting: Story = {
  name: "Live wait and finish",
  parameters: { msw: { handlers: live.handlers } },
  render: () => <Simulation simulation={live} />,
};
