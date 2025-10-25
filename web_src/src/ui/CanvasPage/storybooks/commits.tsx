interface Commit {
  message: string;
  sha: string;
}

export function genCommit(): Commit {
  const index = Math.floor(Math.random() * exampleCommits.length);
  return exampleCommits[index]!;
}

const exampleCommits: Commit[] = [
  { message: "FEAT-1234: Add user authentication", sha: "ef758d40" },
  { message: "FIX-5678: Resolve payment gateway bug", sha: "ab12cd34" },
  { message: "DOCS-9101: Update API documentation", sha: "gh56ij78" },
  { message: "CHORE-1121: Upgrade dependencies", sha: "kl90mn12" },
  { message: "FEAT-2345: Implement user profile settings", sha: "pq34rs56" },
  { message: "FIX-6789: Fix memory leak in worker process", sha: "tu78vw90" },
  { message: "REFACTOR-3456: Simplify database query logic", sha: "xy12za34" },
  { message: "FEAT-4567: Add real-time notifications", sha: "bc56de78" },
  { message: "FIX-7890: Correct timezone calculation", sha: "fg90hi12" },
  { message: "DOCS-1234: Add deployment guide", sha: "jk34lm56" },
  { message: "CHORE-5678: Update CI/CD pipeline", sha: "no78pq90" },
  { message: "FEAT-8901: Implement data export feature", sha: "rs12tu34" },
  { message: "PERF-6789: Optimize database indexes", sha: "za90bc12" },
  { message: "FEAT-3456: Add dashboard analytics", sha: "de34fg56" },
  { message: "FIX-7890: Resolve race condition in sync", sha: "hi78jk90" },
  { message: "STYLE-1234: Update component styling", sha: "lm12no34" },
  { message: "FEAT-4567: Implement file upload system", sha: "pq56rs78" },
  { message: "FIX-8901: Fix pagination boundary error", sha: "tu90vw12" },
  { message: "DOCS-2345: Add troubleshooting section", sha: "xy34za56" },
  { message: "CHORE-5678: Clean up unused dependencies", sha: "bc78de90" },
  { message: "FEAT-9012: Add user role management", sha: "fg12hi34" },
  { message: "FIX-3456: Correct API response format", sha: "jk56lm78" },
  { message: "TEST-7890: Add integration test suite", sha: "no90pq12" },
  { message: "FEAT-1357: Implement search functionality", sha: "rs34tu56" },
  { message: "FIX-2468: Fix dropdown menu alignment", sha: "vw78xy90" },
  { message: "REFACTOR-9753: Extract common utilities", sha: "ab12cd78" },
  { message: "FEAT-8642: Add email notification system", sha: "ef34gh90" },
  { message: "FIX-1975: Resolve mobile responsive issues", sha: "ij56kl12" },
  { message: "PERF-3691: Improve query performance", sha: "mn78op34" },
  { message: "FEAT-7418: Implement audit logging", sha: "qr90st56" },
  { message: "FIX-5284: Fix session timeout handling", sha: "uv12wx78" },
  { message: "DOCS-6397: Update installation instructions", sha: "yz34ab90" },
  { message: "CHORE-4826: Update security dependencies", sha: "cd56ef12" },
];
