import fs from "fs";
import path from "path";
import * as yaml from "js-yaml";

fs.mkdirSync("docs", { recursive: true });

// Correct path: compose files live in deploy/stack/
const stackDir = "deploy/stack";
if (!fs.existsSync(stackDir)) {
  console.warn(`Stack directory not found at ${stackDir}. Writing empty docker.json.`);
  fs.writeFileSync("docs/docker.json", "[]");
  process.exit(0);
}

const composeFiles = fs.readdirSync(stackDir)
  .filter(f => f.endsWith("-compose.yml") || f === "docker-compose.yml");

const allServices = [];

for (const file of composeFiles) {
  try {
    const doc = yaml.load(fs.readFileSync(path.join(stackDir, file), "utf8"));
    if (doc && doc.services) {
      Object.entries(doc.services).forEach(([name, svc]) => {
        allServices.push({
          name,
          image: svc.image || "build",
          ports: svc.ports || [],
          stack: file.replace("-compose.yml", ""),
        });
      });
    }
  } catch (e) {
    console.error(`Failed to parse ${file}:`, e.message);
  }
}

fs.writeFileSync("docs/docker.json", JSON.stringify(allServices, null, 2));
console.log(`Parsed ${allServices.length} Docker services from ${composeFiles.length} compose files.`);
