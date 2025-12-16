#!/usr/bin/env node

const fs = require("fs");
const path = require("path");

function ensureDir(p) {
  if (!fs.existsSync(p)) {
    fs.mkdirSync(p, { recursive: true });
  }
}

function renderTemplate(templatePath, outputPath) {
  const content = fs.readFileSync(templatePath, "utf8");
  fs.writeFileSync(outputPath, content, "utf8");
}

function renderTemplateWithVars(templatePath, outputPath, vars) {
  let content = fs.readFileSync(templatePath, "utf8");
  Object.entries(vars).forEach(([key, value]) => {
    const placeholder = new RegExp(`__${key}__`, "g");
    content = content.replace(placeholder, value);
  });
  fs.writeFileSync(outputPath, content, "utf8");
}

function main() {
  const version = process.argv[2];
  if (!version || version.length < 1) {
    console.error("Usage: node release/local/build.js <version>");
    process.exit(1);
  }

  const root = process.cwd();
  const buildPath = path.join(root, "release", "build", `local-${version}`);
  const bundlePath = path.join(buildPath, "superplane");

  console.log(`Building local release in ${buildPath}`);

  ensureDir(bundlePath);

  const templatesDir = path.join(__dirname, "templates");

  const composeTemplate = path.join(templatesDir, "docker-compose.yml.template");
  const startTemplate = path.join(templatesDir, "start.sh.template");
  const readmeTemplate = path.join(templatesDir, "README.md.template");

  const composeOutput = path.join(bundlePath, "docker-compose.yml");
  const startOutput = path.join(bundlePath, "start.sh");
  const readmeOutput = path.join(bundlePath, "README.md");

  const superplaneImage = `us-east4-docker.pkg.dev/superplane-production/superplane-production/superplane:${version}`;

  renderTemplateWithVars(composeTemplate, composeOutput, {
    SUPERPLANE_IMAGE: superplaneImage,
  });
  renderTemplate(startTemplate, startOutput);
  renderTemplate(readmeTemplate, readmeOutput);

  fs.chmodSync(startOutput, 0o755);

  console.log("* Injected docker-compose.yml");
  console.log("* Injected start.sh");
  console.log("* Injected README.md");

  console.log("* Creating tarball superplane.tar.gz");
  const { spawnSync } = require("child_process");
  const result = spawnSync("bash", ["-c", `cd "${buildPath}" && tar czf superplane.tar.gz superplane`], {
    stdio: "inherit",
  });

  if (result.status !== 0) {
    console.error("Failed to create tarball");
    process.exit(result.status || 1);
  }

  console.log("\nLocal trial release built.");
}

if (require.main === module) {
  main();
}
