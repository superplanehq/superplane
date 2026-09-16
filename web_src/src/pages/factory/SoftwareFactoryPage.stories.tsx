import type { Meta, StoryObj } from "@storybook/react-vite";
import { useCallback, useState } from "react";
import { fn } from "storybook/test";

import { Button } from "@/components/ui/button";

import {
  draftEvents,
  implementationAutomation,
  paymentsFactory,
  successfulEvents,
  verificationAutomation,
  workOrders as fixtureWorkOrders,
} from "./__fixtures__/minimalFactoryFixtures";
import type { NewFactoryInput, NewWorkOrderInput, WorkOrder, WorkOrderEvent } from "./factoryTypes";
import { NewSoftwareFactoryDialog } from "./NewSoftwareFactoryDialog";
import { NewWorkOrderPage } from "./NewWorkOrderPage";
import { SoftwareFactoryPage } from "./SoftwareFactoryPage";
import { WorkOrderPage } from "./WorkOrderPage";

const meta = {
  title: "Pages/Software Factory Minimal",
  component: SoftwareFactoryPage,
  parameters: {
    layout: "fullscreen",
  },
} satisfies Meta<typeof SoftwareFactoryPage>;

export default meta;

type Story = StoryObj<typeof meta>;

const openWorkOrder = fn();
const openAutomation = fn();
const createAutomation = fn();
const factoryAutomations = [implementationAutomation, verificationAutomation];

export const Factory: Story = {
  render: () => <InteractiveFactoryStory />,
};

export const NewFactory: Story = {
  args: {
    factory: {
      id: "factory-new",
      name: "Mobile Factory",
      description: "Implementation work for the mobile applications.",
    },
    workOrders: [],
    automations: [],
    currentUserId: "user-darko",
    onNewWorkOrder: fn(),
    onOpenWorkOrder: openWorkOrder,
    onCreateAutomation: createAutomation,
    onOpenAutomation: openAutomation,
  },
};

export const CreateFactory: Story = {
  render: () => <CreateFactoryStory />,
};

export const NewWorkOrder: Story = {
  render: () => (
    <NewWorkOrderPage factory={paymentsFactory} automations={factoryAutomations} onCancel={fn()} onCreate={fn()} />
  ),
};

export const DraftWorkOrder: Story = {
  render: () => <InteractiveDraftWorkOrderStory />,
};

export const SuccessfulWorkOrder: Story = {
  render: () => (
    <WorkOrderPage
      factory={paymentsFactory}
      workOrder={fixtureWorkOrders[3]!}
      events={successfulEvents}
      onBack={fn()}
      onApprove={fn()}
    />
  ),
};

function InteractiveFactoryStory() {
  const [workOrders, setWorkOrders] = useState(fixtureWorkOrders);
  const [isCreatingWorkOrder, setIsCreatingWorkOrder] = useState(false);

  const createWorkOrder = useCallback((input: NewWorkOrderInput) => {
    const createdAt = new Date().toISOString();
    const selectedAutomations = factoryAutomations
      .filter((automation) => input.automationIds.includes(automation.id))
      .map((automation) => ({ id: automation.id, name: automation.name, state: "planned" as const }));
    setWorkOrders((current) => [
      {
        id: `work-order-${current.length + 1}`,
        title: input.title,
        description: input.description,
        state: "draft",
        createdByUserId: "user-darko",
        createdByName: "Darko",
        createdAt,
        updatedAt: createdAt,
        automations: selectedAutomations,
      },
      ...current,
    ]);
    setIsCreatingWorkOrder(false);
  }, []);

  const openNewWorkOrder = useCallback(() => setIsCreatingWorkOrder(true), []);
  const cancelNewWorkOrder = useCallback(() => setIsCreatingWorkOrder(false), []);

  if (isCreatingWorkOrder) {
    return (
      <NewWorkOrderPage
        factory={paymentsFactory}
        automations={factoryAutomations}
        onCancel={cancelNewWorkOrder}
        onCreate={createWorkOrder}
      />
    );
  }

  return (
    <SoftwareFactoryPage
      factory={paymentsFactory}
      workOrders={workOrders}
      automations={factoryAutomations}
      currentUserId="user-darko"
      onNewWorkOrder={openNewWorkOrder}
      onOpenWorkOrder={openWorkOrder}
      onCreateAutomation={createAutomation}
      onOpenAutomation={openAutomation}
    />
  );
}

function CreateFactoryStory() {
  const [isOpen, setIsOpen] = useState(true);
  const [createdFactory, setCreatedFactory] = useState<NewFactoryInput>();

  return (
    <div className="flex min-h-screen items-center justify-center bg-slate-100 p-6 dark:bg-gray-950">
      <div className="text-center">
        <p className="text-sm text-slate-600 dark:text-gray-300">
          {createdFactory ? `${createdFactory.name} created` : "Create a first-class Factory workspace."}
        </p>
        <Button type="button" className="mt-4" onClick={() => setIsOpen(true)}>
          New Software Factory
        </Button>
      </div>
      <NewSoftwareFactoryDialog open={isOpen} onOpenChange={setIsOpen} onCreate={setCreatedFactory} />
    </div>
  );
}

function InteractiveDraftWorkOrderStory() {
  const [workOrder, setWorkOrder] = useState<WorkOrder>(fixtureWorkOrders[0]!);
  const [events, setEvents] = useState<WorkOrderEvent[]>(draftEvents);

  const approveWorkOrder = useCallback(() => {
    const occurredAt = new Date().toISOString();
    setWorkOrder((current) => ({
      ...current,
      state: "ready",
      updatedAt: occurredAt,
    }));
    setEvents((current) => [
      ...current,
      {
        id: `event-${current.length + 1}`,
        kind: "approved",
        summary: "Approved and moved to ready",
        actor: "Darko",
        occurredAt,
      },
    ]);
  }, []);

  return (
    <WorkOrderPage
      factory={paymentsFactory}
      workOrder={workOrder}
      events={events}
      onBack={fn()}
      onApprove={approveWorkOrder}
    />
  );
}
