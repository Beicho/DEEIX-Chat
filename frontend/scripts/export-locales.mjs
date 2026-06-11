import { cp, rm } from "node:fs/promises";
import { join } from "node:path";
import { spawn } from "node:child_process";

const locales = ["zh-CN", "en-US"];
const projectRoot = process.cwd();
const outDir = join(projectRoot, "out");
const localeOutDir = join(projectRoot, "out-locales");

function runBuild(locale) {
  return new Promise((resolve, reject) => {
    const child = spawn("pnpm", ["exec", "next", "build", "--webpack"], {
      cwd: projectRoot,
      env: {
        ...process.env,
        NEXT_PUBLIC_BUILD_LOCALE: locale,
      },
      stdio: "inherit",
    });
    child.on("error", reject);
    child.on("exit", (code) => {
      if (code === 0) {
        resolve();
        return;
      }
      reject(new Error(`next build failed for ${locale} with exit code ${code ?? "unknown"}`));
    });
  });
}

await rm(localeOutDir, { recursive: true, force: true });

for (const locale of locales) {
  await rm(outDir, { recursive: true, force: true });
  await runBuild(locale);
  await cp(outDir, join(localeOutDir, locale), { recursive: true });
}

await rm(outDir, { recursive: true, force: true });
await cp(join(localeOutDir, "zh-CN"), outDir, { recursive: true });
for (const locale of locales) {
  await cp(join(localeOutDir, locale), join(outDir, locale), { recursive: true });
}
await rm(localeOutDir, { recursive: true, force: true });
