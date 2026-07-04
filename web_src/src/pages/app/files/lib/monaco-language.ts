const EXTENSION_TO_MONACO_LANGUAGE: Record<string, string> = {
  css: "css",
  go: "go",
  html: "html",
  js: "javascript",
  mjs: "javascript",
  cjs: "javascript",
  json: "json",
  jsonc: "json",
  jsx: "javascript",
  md: "markdown",
  mdx: "markdown",
  markdown: "markdown",
  py: "python",
  sh: "shell",
  bash: "shell",
  zsh: "shell",
  ts: "typescript",
  tsx: "typescript",
  xml: "xml",
  yaml: "yaml",
  yml: "yaml",
};

export function getFileMonacoLanguage(path: string): string {
  const normalizedPath = path.toLowerCase();

  if (normalizedPath.endsWith("dockerfile") || normalizedPath.includes("/dockerfile")) {
    return "dockerfile";
  }

  if (normalizedPath.endsWith("makefile") || normalizedPath.includes("/makefile")) {
    return "makefile";
  }

  const extension = normalizedPath.split(".").pop();
  if (!extension) {
    return "plaintext";
  }

  return EXTENSION_TO_MONACO_LANGUAGE[extension] ?? "plaintext";
}
