import { usePageTitle } from "@/hooks/usePageTitle";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import { HomePageShell } from "./HomePageShell";
import { ZeroStatePage } from "./ZeroStatePage";

export function NewAppPage() {
  usePageTitle(["New App"]);
  useReportPageReady(true);

  return (
    <HomePageShell>
      <ZeroStatePage />
    </HomePageShell>
  );
}
