import fs from "fs";

if (!fs.existsSync("docs")) {
  fs.mkdirSync("docs");
}

const file = fs.existsSync(".env.example") ? ".env.example" : ".env";
if (!fs.existsSync(file)) {
  console.log("No env file found.");
  process.exit(0);
}

const lines = fs.readFileSync(file, "utf-8").split("\n");
const vars = [];

lines.forEach(line => {
  const l = line.trim();
  if (l && !l.startsWith("#") && l.includes("=")) {
    const [key, ...rest] = l.split("=");
    vars.push({ 
      key: key.trim(), 
      default: rest.join("=").trim() 
    });
  }
});

fs.writeFileSync("docs/env.json", JSON.stringify(vars, null, 2));
console.log(`Parsed ${vars.length} environment variables.`);
