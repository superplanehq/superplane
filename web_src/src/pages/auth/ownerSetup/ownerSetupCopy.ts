export const OWNER_SETUP_COPY = {
  smtp: {
    headline: "Send email from this installation",
    body: "SMTP sends invites and magic codes. The owner password login works without it.",
    save: "Save and continue",
    saving: "Saving…",
    skip: "Skip for now",
    host: "SMTP host",
    port: "SMTP port",
    username: "SMTP username",
    password: "SMTP password",
    fromName: "From name",
    fromEmail: "From email",
    useTls: "Use TLS (STARTTLS)",
    useTlsHelp: "Encrypt the connection to the SMTP server.",
    required: "Enter the SMTP host, port, and from email.",
    error: "SuperPlane could not save SMTP settings. Check the values and try again.",
  },
  fleet: {
    headline: "Set up runner fleets",
    body: "Use the standalone Fleet Manager chart to connect runner infrastructure to this SuperPlane installation.",
    helper:
      "The reusable setup page generates values for AWS, GCP, or Azure. It creates a token only when you request installation instructions.",
    open: "Open Fleet Manager setup",
    continue: "Finish installation",
    skip: "Skip for now",
  },
} as const;
