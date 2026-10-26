import { useEffect, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import type { OnboardingCardSession } from "../onboardingCardPlan";
import { FirstRunHeading, FirstRunShell } from "./FirstRunShell";
import type { FirstRunChrome } from "./firstRunTypes";
import type { FirstRunSphereProps } from "./FirstRunSpherePane";

export function FirstRunCardForm({
  session,
  chrome,
  sphere,
  onSaved,
  onBack,
  onRetry,
}: {
  session: OnboardingCardSession;
  chrome: FirstRunChrome;
  sphere?: FirstRunSphereProps;
  onSaved: () => void;
  onBack: () => void;
  onRetry: () => void;
}) {
  const frame = useRef<HTMLIFrameElement>(null);
  const saved = useRef(false);
  const [height, setHeight] = useState(0);
  const [loaded, setLoaded] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");
  const url = new URL("/embed/payment-method", session.origin);
  url.searchParams.set("session_token", session.token);
  url.searchParams.set("embed", "true");
  url.searchParams.set("embed_origin", window.location.origin);
  url.searchParams.set("theme", "dark");
  url.searchParams.set("set_default", "true");

  useEffect(() => {
    // Polar's SDK currently fixes this URL to production. Use the same
    // iframe protocol here so sandbox sessions stay on the sandbox host.
    const timeout = window.setTimeout(() => {
      setError("Could not load the secure card form. Try again, or contact your organization admin.");
    }, 15000);
    const receive = (event: MessageEvent) => {
      if (event.origin !== session.origin || event.source !== frame.current?.contentWindow) return;
      const data = event.data;
      if (!isPaymentMethodMessage(data)) return;
      switch (data.event) {
        case "loaded":
          window.clearTimeout(timeout);
          setLoaded(true);
          setError("");
          break;
        case "resize":
          if (typeof data.height === "number" && Number.isFinite(data.height))
            setHeight(Math.min(2000, Math.max(0, Math.ceil(data.height))));
          break;
        case "confirmed":
          setSubmitting(true);
          setError("");
          break;
        case "success":
          if (saved.current || typeof data.paymentMethodId !== "string" || !data.paymentMethodId) return;
          saved.current = true;
          window.clearTimeout(timeout);
          onSaved();
          break;
        case "error":
          window.clearTimeout(timeout);
          setSubmitting(false);
          setError("Could not save your card. Check the card details or try again.");
          break;
      }
    };
    window.addEventListener("message", receive);
    return () => {
      window.clearTimeout(timeout);
      window.removeEventListener("message", receive);
    };
  }, [session.origin, onSaved]);

  return (
    <FirstRunShell
      testId="first-run-card-form"
      chrome={{ ...chrome, onBack }}
      busy={submitting}
      sphere={sphere}
      visual="preview"
    >
      <FirstRunHeading headline="Add your card">
        <p>No subscription charge on Pay-as-you-go.</p>
      </FirstRunHeading>
      {!loaded && !error && (
        <p role="status" className="mt-6 text-sm text-muted-foreground">
          Loading secure card form…
        </p>
      )}
      <iframe
        ref={frame}
        title="Secure card setup"
        src={url.toString()}
        className="mt-6 block w-full border-0"
        style={{ height, colorScheme: "dark" }}
        scrolling="no"
        allow={`payment 'self' ${session.origin}; publickey-credentials-get 'self' ${session.origin};`}
      />
      {error && (
        <div className="mt-4 space-y-3">
          <p role="alert" className="text-sm text-destructive">
            {error}
          </p>
          <Button variant="outline" onClick={onRetry}>
            Try again
          </Button>
        </div>
      )}
    </FirstRunShell>
  );
}

function isPaymentMethodMessage(data: unknown): data is {
  type: "POLAR_PAYMENT_METHOD";
  event: string;
  height?: unknown;
  paymentMethodId?: unknown;
} {
  return (
    typeof data === "object" &&
    data !== null &&
    "type" in data &&
    data.type === "POLAR_PAYMENT_METHOD" &&
    "event" in data &&
    typeof data.event === "string"
  );
}
