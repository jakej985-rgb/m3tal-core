import fs from "fs";
import path from "path";

fs.mkdirSync("docs", { recursive: true });

// Scan the actual Go source layout: cmd/ and internal/ packages
const modules = [];

const scanDir = (dir, type) => {
  const basePath = path.resolve(".");
  const joinedPath = path.join(basePath, dir);
  const fullPath = path.normalize(joinedPath);

  if (!fullPath.startsWith(basePath)) return;

  if (!fs.existsSync(fullPath)) return;
  fs.readdirSync(fullPath, { withFileTypes: true })
    .filter(d => d.isDirectory() && !d.name.startsWith(".") && d.name !== "node_modules" && d.name !== "cmd")
    .forEach(d => {
      modules.push({
        name: d.name,
        type,
        path: `${dir}/${d.name}`
      });
    });
};

// Core Go binary entrypoints
if (fs.existsSync("cli")) {
  modules.push({
    name: "m3tal (cli)",
    type: "CLI binary",
    path: "cli"
  });
}
if (fs.existsSync("api/cmd")) {
  modules.push({
    name: "m3tal-api",
    type: "API binary",
    path: "api/cmd"
  });
}

// Core Go packages
scanDir("core", "Core package");

// Shared Go packages
scanDir("pkg", "Shared package");

// API framework files / handlers
scanDir("api", "API package");

// UI layers
if (fs.existsSync("tui")) {
  modules.push({
    name: "tui",
    type: "Terminal UI",
    path: "tui"
  });
}
if (fs.existsSync("webui")) {
  modules.push({
    name: "webui",
    type: "Web UI",
    path: "webui"
  });
}
if (fs.existsSync("gui")) {
  modules.push({
    name: "gui",
    type: "Desktop UI",
    path: "gui"
  });
}

// Deploy artifacts
if (fs.existsSync("deploy")) {
  fs.readdirSync("deploy", { withFileTypes: true })
    .filter(d => d.isDirectory() && !d.name.startsWith("."))
    .forEach(d => {
      modules.push({
        name: d.name,
        type: "Deploy artifact",
        path: `deploy/${d.name}`
      });
    });
}

// Top-level markers
if (fs.existsSync("go.mod")) {
  const modContent = fs.readFileSync("go.mod", "utf-8");
  const modLine = modContent.split("\n").find(l => l.startsWith("module "));
  modules.push({
    name: modLine ? modLine.replace("module ", "").trim() : "go-module",
    type: "Go module",
    path: "go.mod"
  });
}

fs.writeFileSync("docs/m3tal-services.json", JSON.stringify(modules, null, 2));
console.log(`Discovered ${modules.length} M3TAL source modules.`);
