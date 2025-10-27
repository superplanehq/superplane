interface DockerImage {
  message: string;
  sha: string;
  size: string;
}

export function genDockerImage(): DockerImage {
  const index = Math.floor(Math.random() * examplePushes.length);
  return examplePushes[index]!;
}

const examplePushes: DockerImage[] = [
  { message: "v1.0.1", sha: "ef758d40", size: "150MB" },
  { message: "v1.0.2", sha: "ab12cd34", size: "155MB" },
  { message: "v1.1.0", sha: "gh56ij78", size: "200MB" },
  { message: "v1.1.1", sha: "kl90mn12", size: "205MB" },
  { message: "v1.2.0", sha: "pq34rs56", size: "250MB" },
  { message: "v1.2.1", sha: "tu78vw90", size: "255MB" },
  { message: "v2.0.0", sha: "xy12za34", size: "300MB" },
  { message: "v2.0.1", sha: "bc56de78", size: "305MB" },
  { message: "v2.1.0", sha: "fg90hi12", size: "350MB" },
  { message: "v2.1.1", sha: "jk34lm56", size: "355MB" },
  { message: "v2.2.0", sha: "no78pq90", size: "400MB" },
  { message: "v2.2.1", sha: "rs12tu34", size: "405MB" },
  { message: "v3.0.0", sha: "za90bc12", size: "450MB" },
  { message: "v3.0.1", sha: "de34fg56", size: "455MB" },
  { message: "v3.1.0", sha: "hi78jk90", size: "500MB" },
  { message: "v3.1.1", sha: "lm12no34", size: "505MB" },
  { message: "v3.2.0", sha: "pq56rs78", size: "550MB" },
  { message: "v3.2.1", sha: "tu90vw12", size: "555MB" },
  { message: "v4.0.0", sha: "xy34za56", size: "600MB" },
  { message: "v4.0.1", sha: "bc78de90", size: "605MB" },
  { message: "v4.1.0", sha: "fg12hi34", size: "650MB" },
  { message: "v4.1.1", sha: "jk56lm78", size: "655MB" },
  { message: "v4.2.0", sha: "no90pq12", size: "700MB" },
  { message: "v4.2.1", sha: "rs34tu56", size: "705MB" },
  { message: "v5.0.0", sha: "vw78xy90", size: "750MB" },
  { message: "v5.0.1", sha: "ab12cd78", size: "755MB" },
];
