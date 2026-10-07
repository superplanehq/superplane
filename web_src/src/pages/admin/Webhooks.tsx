import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useSearchParams } from "react-router";

import { DatadogWebhooks } from "./DatadogWebhooks";
import { LinearWebhooks } from "./LinearWebhooks";
import { SentryWebhooks } from "./SentryWebhooks";

const WEBHOOKS_TITLE = "Webhooks";
const WEBHOOK_SERVICES = ["sentry", "datadog", "linear"] as const;

type WebhookService = (typeof WEBHOOK_SERVICES)[number];

function isWebhookService(value: string | null): value is WebhookService {
  return WEBHOOK_SERVICES.some((service) => service === value);
}

export function Webhooks() {
  const [searchParams, setSearchParams] = useSearchParams();
  const requested = searchParams.get("service");
  const service: WebhookService = isWebhookService(requested) ? requested : "sentry";

  return (
    <div className="space-y-6">
      <h1 className="text-xl font-semibold text-gray-900 dark:text-gray-100">{WEBHOOKS_TITLE}</h1>
      <Tabs
        value={service}
        onValueChange={(next) => {
          if (!isWebhookService(next)) {
            return;
          }
          if (next === "sentry") {
            setSearchParams({});
            return;
          }
          setSearchParams({ service: next });
        }}
      >
        <TabsList>
          <TabsTrigger value="sentry">Sentry</TabsTrigger>
          <TabsTrigger value="datadog">Datadog</TabsTrigger>
          <TabsTrigger value="linear">Linear</TabsTrigger>
        </TabsList>
        <TabsContent value="sentry" className="mt-4">
          <SentryWebhooks />
        </TabsContent>
        <TabsContent value="datadog" className="mt-4">
          <DatadogWebhooks />
        </TabsContent>
        <TabsContent value="linear" className="mt-4">
          <LinearWebhooks />
        </TabsContent>
      </Tabs>
    </div>
  );
}
