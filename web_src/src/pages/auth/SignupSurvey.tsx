import React, { useState } from "react";
import { Alert, AlertDescription, AlertTitle } from "@/ui/alert";
import { LoadingButton } from "@/components/ui/loading-button";
import { Textarea } from "@/components/ui/textarea";
import { Text } from "../../components/Text/text";

type SourceChannel = "search" | "social" | "referral" | "content" | "event" | "partner" | "other";

type Role = "engineer" | "devops" | "manager" | "founder" | "product" | "other";

const SOURCE_OPTIONS: { value: SourceChannel; label: string }[] = [
  { value: "search", label: "Search engine (Google, etc.)" },
  { value: "social", label: "Social media (X, LinkedIn, YouTube, Reddit)" },
  { value: "referral", label: "Referred by a colleague or friend" },
  { value: "content", label: "Blog, newsletter, or podcast" },
  { value: "event", label: "Conference or event" },
  { value: "partner", label: "Partner or integration (e.g., Semaphore)" },
  { value: "other", label: "Other" },
];

const ROLE_OPTIONS: { value: Role; label: string }[] = [
  { value: "engineer", label: "Engineer / Developer" },
  { value: "devops", label: "DevOps / Platform / SRE" },
  { value: "manager", label: "Engineering manager" },
  { value: "founder", label: "Founder / Executive" },
  { value: "product", label: "Product manager" },
  { value: "other", label: "Other" },
];

// nextDestination returns the URL to redirect to after the survey is
// submitted or skipped. It reads `?next=` from the current location and
// restricts it to same-origin paths so the survey can't be weaponised as an
// open redirect.
function nextDestination(): string {
  const raw = new URLSearchParams(window.location.search).get("next");
  if (raw && raw.startsWith("/") && !raw.startsWith("//")) return raw;
  return "/";
}

const SignupSurvey: React.FC = () => {
  const [source, setSource] = useState<SourceChannel | "">("");
  const [sourceOther, setSourceOther] = useState("");
  const [role, setRole] = useState<Role | "">("");
  const [useCase, setUseCase] = useState("");
  const [loading, setLoading] = useState<"continue" | "skip" | null>(null);
  const [error, setError] = useState<string | null>(null);

  async function submit(payload: Record<string, unknown>) {
    const response = await fetch("/signup-survey", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      body: JSON.stringify(payload),
    });
    if (!response.ok) throw new Error(`status ${response.status}`);
  }

  async function onContinue(e: React.FormEvent) {
    e.preventDefault();
    if (!source) return;
    setError(null);
    setLoading("continue");
    try {
      const body: Record<string, unknown> = {
        skipped: false,
        source_channel: source,
      };
      if (source === "other" && sourceOther.trim()) body.source_other = sourceOther.trim();
      if (role) body.role = role;
      if (useCase.trim()) body.use_case = useCase.trim();

      await submit(body);
      window.location.href = nextDestination();
    } catch {
      setError("We couldn't save your answers. Please try again.");
    } finally {
      setLoading(null);
    }
  }

  async function onSkip() {
    setError(null);
    setLoading("skip");
    try {
      await submit({ skipped: true });
      window.location.href = nextDestination();
    } catch {
      setError("We couldn't save your answers. Please try again.");
    } finally {
      setLoading(null);
    }
  }

  return (
    <div className="min-h-screen bg-slate-100">
      <div className="flex items-center justify-center p-8">
        <div className="max-w-xl w-full bg-white rounded-lg shadow-sm p-8 outline outline-slate-950/10">
          <div className="text-center mb-6">
            <h4 className="text-xl font-semibold text-gray-800 mb-1">Welcome to SuperPlane</h4>
            <Text className="text-gray-800">Help us tailor your experience. Takes about 15 seconds.</Text>
          </div>

          {error ? (
            <Alert variant="destructive" className="mb-4">
              <AlertTitle>Something went wrong</AlertTitle>
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          ) : null}

          <form onSubmit={onContinue} className="space-y-6">
            <fieldset>
              <legend className="block text-sm font-medium text-gray-800 mb-2">
                How did you hear about SuperPlane?
              </legend>
              <div className="space-y-2">
                {SOURCE_OPTIONS.map((opt) => (
                  <label key={opt.value} className="flex items-center gap-2 text-sm text-gray-800">
                    <input
                      type="radio"
                      name="source_channel"
                      value={opt.value}
                      checked={source === opt.value}
                      onChange={() => setSource(opt.value)}
                    />
                    <span>{opt.label}</span>
                  </label>
                ))}
              </div>
              {source === "other" ? (
                <input
                  type="text"
                  value={sourceOther}
                  onChange={(e) => setSourceOther(e.target.value)}
                  placeholder="Tell us where"
                  maxLength={200}
                  className="mt-2 w-full px-3 py-2 outline-1 outline-slate-300 rounded-md shadow-md focus:outline-gray-800"
                />
              ) : null}
            </fieldset>

            <div>
              <label className="block text-sm font-medium text-gray-800 mb-2">
                What's your role? <span className="text-gray-500">(optional)</span>
              </label>
              <div className="flex flex-wrap gap-2">
                {ROLE_OPTIONS.map((opt) => {
                  const active = role === opt.value;
                  return (
                    <button
                      key={opt.value}
                      type="button"
                      onClick={() => setRole(active ? "" : opt.value)}
                      aria-pressed={active}
                      className={
                        "px-3 py-1.5 rounded-full text-sm border transition-colors " +
                        (active
                          ? "bg-gray-800 text-white border-gray-800"
                          : "bg-white text-gray-800 border-slate-300 hover:bg-slate-50")
                      }
                    >
                      {opt.label}
                    </button>
                  );
                })}
              </div>
            </div>

            <div>
              <label htmlFor="use_case" className="block text-sm font-medium text-gray-800 mb-2">
                What do you want to use SuperPlane for? <span className="text-gray-500">(optional)</span>
              </label>
              <Textarea
                id="use_case"
                value={useCase}
                onChange={(e) => setUseCase(e.target.value)}
                placeholder="e.g., deploy our ML models on a schedule"
                maxLength={500}
                rows={2}
              />
            </div>

            <div className="flex flex-col-reverse sm:flex-row gap-3 pt-2">
              <LoadingButton
                type="button"
                onClick={onSkip}
                disabled={loading !== null}
                loadingText="Skipping..."
                className="flex-1 px-4 py-2 text-sm font-medium text-gray-700 rounded-md border border-slate-300 bg-white hover:bg-slate-50 disabled:opacity-60"
              >
                Skip for now
              </LoadingButton>
              <LoadingButton
                type="submit"
                className="flex-1"
                disabled={!source || loading !== null}
                loading={loading === "continue"}
                loadingText="Saving…"
              >
                Continue
              </LoadingButton>
            </div>
          </form>
        </div>
      </div>
    </div>
  );
};

export default SignupSurvey;
