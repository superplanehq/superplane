import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";
import type { OrganizationsPreviewSeatPurchaseResponse, OrganizationsSeatPurchase } from "@/api-client";
import { Button } from "@/components/ui/button";
import type { UseSeatPurchaseResult } from "@/hooks/useSeatPurchase";
import { ACTIVE_BILLING_PERIOD_ENDS_AT } from "../../__fixtures__/usageReportFixtures";
import { BillingSeatPurchaseDialog, type BillingSeatPurchaseDialogProps } from "./BillingSeatPurchaseDialog";

const preview: OrganizationsPreviewSeatPurchaseResponse = {
  purchasedSeats: 10,
  targetSeats: 11,
  seatPriceCents: "2900",
  proratedChargeCents: "1234",
  periodStart: new Date().toISOString(),
  periodEnd: ACTIVE_BILLING_PERIOD_ENDS_AT,
  newMonthlyTotalCents: "31900",
  purchasableSeats: 40,
  estimate: true,
  excludesTax: true,
};

const ready: UseSeatPurchaseResult = {
  isOpen: true,
  setOpen: (open: boolean) => console.log("Change purchase dialog", { open }),
  additionalSeats: "1",
  setAdditionalSeats: (value: string) => console.log("Change additional seats", { value }),
  requestPreview: () => console.log("Request seat purchase estimate"),
  preview,
  previewPending: false,
  previewError: null,
  purchase: undefined,
  purchaseMessage: null,
  attemptSucceeded: false,
  submissionUnresolved: false,
  timedOut: false,
  refreshing: false,
  refreshBilling: async () => console.log("Refresh billing"),
  submitting: false,
  canConfirm: true,
  confirm: async () => console.log("Confirm purchase"),
};

const pendingPurchase: OrganizationsSeatPurchase = {
  id: "storybook-seat-purchase",
  status: "pending",
  fromSeats: 10,
  toSeats: 11,
  createdAt: new Date().toISOString(),
};

function PurchaseDialogStory(args: BillingSeatPurchaseDialogProps) {
  const [isOpen, setOpen] = useState(args.purchase.isOpen);
  const [additionalSeats, setAdditionalSeats] = useState(args.purchase.additionalSeats);
  return (
    <>
      <Button onClick={() => setOpen(true)}>Buy seats</Button>
      <BillingSeatPurchaseDialog
        {...args}
        purchase={{
          ...args.purchase,
          isOpen,
          setOpen: (open: boolean) => {
            args.purchase.setOpen(open);
            setOpen(open);
          },
          additionalSeats,
          preview: additionalSeats === args.purchase.additionalSeats ? args.purchase.preview : undefined,
          canConfirm: args.purchase.canConfirm && additionalSeats === args.purchase.additionalSeats,
          setAdditionalSeats: (value: string) => {
            args.purchase.setAdditionalSeats(value);
            setAdditionalSeats(value);
          },
        }}
      />
    </>
  );
}

const meta = {
  title: "Factories/Settings/Buy seats",
  component: BillingSeatPurchaseDialog,
  render: (args) => <PurchaseDialogStory {...args} />,
  args: {
    purchase: ready,
    purchasableSeats: 40,
    planMaximumSeats: 50,
    cancelAtPeriodEnd: false,
    paymentPending: false,
    onUpdatePaymentMethod: () => console.log("Update payment method"),
  },
} satisfies Meta<typeof BillingSeatPurchaseDialog>;

export default meta;
type Story = StoryObj<typeof meta>;

export const PreviewReady: Story = {};

export const PreviewLoading: Story = {
  args: { purchase: { ...ready, preview: undefined, previewPending: true, canConfirm: false } },
};

export const InvalidCount: Story = {
  args: { purchase: { ...ready, additionalSeats: "1.5", preview: undefined, canConfirm: false } },
};

export const Submitting: Story = {
  args: { purchase: { ...ready, submitting: true, canConfirm: false } },
};

export const Succeeded: Story = {
  args: {
    purchase: {
      ...ready,
      preview: undefined,
      purchase: { ...pendingPurchase, status: "succeeded" },
      purchaseMessage: "Seats added.",
      attemptSucceeded: true,
      canConfirm: false,
    },
  },
};

export const SubmissionUnresolved: Story = {
  args: {
    purchase: {
      ...ready,
      preview: undefined,
      purchaseMessage: "SuperPlane could not complete this purchase. Refresh billing to check its status.",
      submissionUnresolved: true,
      canConfirm: false,
    },
  },
};

export const Failed: Story = {
  args: {
    purchase: {
      ...ready,
      preview: undefined,
      purchase: { ...pendingPurchase, status: "failed", message: "The card was declined. Update the payment method." },
      canConfirm: false,
    },
  },
};

export const BankConfirmationRequired: Story = {
  args: {
    purchase: {
      ...ready,
      preview: undefined,
      purchase: {
        ...pendingPurchase,
        status: "action_required",
        message: "The bank requires payment confirmation. Update the payment method to continue.",
      },
      canConfirm: false,
    },
  },
};

export const Pending: Story = {
  args: { purchase: { ...ready, preview: undefined, purchase: pendingPurchase, canConfirm: false } },
};

export const AwaitingConfirmation: Story = {
  args: {
    purchase: {
      ...ready,
      preview: undefined,
      purchase: { ...pendingPurchase, status: "awaiting_confirmation" },
      canConfirm: false,
    },
  },
};

export const ConfirmationTimeout: Story = {
  args: { purchase: { ...ready, preview: undefined, purchase: pendingPurchase, timedOut: true, canConfirm: false } },
};

export const PlanMaximum: Story = {
  args: { purchasableSeats: 0, purchase: { ...ready, preview: undefined, canConfirm: false } },
};

export const SubscriptionCanceled: Story = {
  args: { cancelAtPeriodEnd: true, purchase: { ...ready, preview: undefined, canConfirm: false } },
};

export const RenewalFreeze: Story = {
  args: {
    purchase: {
      ...ready,
      preview: undefined,
      previewError: "Seat purchases are unavailable during renewal. Try again after renewal.",
      canConfirm: false,
    },
  },
};

export const UpdatingPaymentMethod: Story = {
  ...Failed,
  args: { ...Failed.args, paymentPending: true },
};
