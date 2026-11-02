import { Button } from "@/components/ui/button";

export function LoadError({ retry, subject }: { retry: () => void; subject: string }) {
  return (
    <div role="alert" className="space-y-2">
      <p>Could not load {subject}. Check access and retry.</p>
      <Button variant="outline" onClick={retry}>
        Retry
      </Button>
    </div>
  );
}
