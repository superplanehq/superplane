import { useState } from "react";
import { Button } from "@/components/ui/button";
import { DialogFooter } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

export function ProviderKeyStep({
  providerName,
  onBack,
  onSaveKey,
}: {
  providerName: string;
  onBack: () => void;
  onSaveKey: (apiKey: string) => void;
}) {
  const [apiKey, setApiKey] = useState("");

  return (
    <form
      className="space-y-4"
      onSubmit={(event) => {
        event.preventDefault();
        if (apiKey.trim() === "") return;
        onSaveKey(apiKey.trim());
      }}
    >
      <div className="space-y-2">
        <Label htmlFor="llm-models-switch-api-key">{providerName} API key</Label>
        <Input
          id="llm-models-switch-api-key"
          data-testid="llm-models-switch-api-key"
          type="password"
          autoComplete="off"
          value={apiKey}
          onChange={(event) => setApiKey(event.target.value)}
        />
      </div>
      <DialogFooter>
        <Button type="button" variant="outline" onClick={onBack}>
          Back
        </Button>
        <Button type="submit" disabled={apiKey.trim() === ""}>
          Save and switch
        </Button>
      </DialogFooter>
    </form>
  );
}
