import fs from "fs";

function readJSON(path) {
  return fs.existsSync(path) ? JSON.parse(fs.readFileSync(path)) : [];
}

const docker = readJSON("docs/docker.json");
const env = readJSON("docs/env.json");
const services = readJSON("docs/m3tal-services.json");

let md = `# M3TAL Media Server\n\n`;

md += `Autonomous, self-healing media automation platform.\n\n`;

md += `## 🏗 Architecture (Source Services)\n`;
md += `| Service | Type | Path |\n|---------|------|------|\n`;
services.forEach(s => {
  md += `| ${s.name} | ${s.type} | [${s.path}](./${s.path}) |\n`;
});

if (docker.length) {
  md += `\n## 🐳 Infrastructure Stacks\n`;
  // Group by stack
  const stacks = {};
  docker.forEach(s => {
    if (!stacks[s.stack]) stacks[s.stack] = [];
    stacks[s.stack].push(s);
  });

  Object.entries(stacks).forEach(([stackName, svcList]) => {
    md += `### ${stackName.charAt(0).toUpperCase() + stackName.slice(1)} Stack\n`;
    svcList.forEach(s => {
      md += `- **${s.name}** (${s.image})\n`;
      if (s.ports.length) {
        md += `  - Ports: ${s.ports.join(", ")}\n`;
      }
    });
    md += `\n`;
  });
}

if (env.length) {
  md += `## ⚙️ Environment Configuration\n`;
  md += `| Variable | Default |\n|----------|---------|\n`;
  env.forEach(v => {
    // Escape underscores for markdown table
    const key = v.key.replace(/_/g, "\\_");
    const val = v.default.replace(/_/g, "\\_");
    md += `| ${key} | \`${val || "N/A"}\` |\n`;
  });
}

md += `\n## 🚀 Deployment\n\n`;
md += `\`\`\`bash\npython m3tal.py init\n\`\`\`\n`;

fs.writeFileSync("README.generated.md", md);
console.log("README.generated.md assembled.");
