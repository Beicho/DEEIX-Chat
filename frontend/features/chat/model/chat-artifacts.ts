import type { ChatAreaMessage } from "@/features/chat/types/messages";
import {
  resolveArtifactPreviewKind,
  type ArtifactPreviewKind,
} from "@/shared/lib/artifact-preview";

export type { ArtifactPreviewKind } from "@/shared/lib/artifact-preview";

export type ChatArtifact = {
  id: string;
  messageID: string;
  messageKey: string;
  runID?: string;
  blockIndex: number;
  kind: ArtifactPreviewKind;
  language: string;
  code: string;
  complete: boolean;
  streaming: boolean;
  updatedAt?: string;
};

export type OpenCodeArtifactInput = {
  code: string;
  language: string;
  kind: ArtifactPreviewKind;
};

export type ArtifactPreviewLabels = {
  htmlTitle: string;
  cssTitle: string;
  cssEyebrow: string;
  cssHeading: string;
  cssDescription: string;
  cssPrimaryAction: string;
  cssSecondaryAction: string;
  cssCardTitle: string;
  cssCardDescription: string;
  cssMetricTitle: string;
  jsTitle: string;
  svgTitle: string;
  markdownTitle: string;
  mermaidTitle: string;
  reactTitle: string;
  unknownError: string;
  reactMissingComponent: string;
  reactUnsupportedImport: string;
};

const SCRIPT_CLOSE_RE = /<\/script/gi;
const STYLE_CLOSE_RE = /<\/style/gi;
const FENCE_OPEN_RE = /^[ \t]*(`{3,}|~{3,})([^\n]*)$/;
const DOCTYPE_RE = /<!doctype\s+html[^>]*>/i;
const HTML_OPEN_RE = /<html\b[^>]*>/i;
const HTML_CLOSE_RE = /<\/html\s*>/i;
const HEAD_BLOCK_RE = /<head\b[^>]*>([\s\S]*?)<\/head\s*>/i;
const BODY_BLOCK_RE = /<body\b[^>]*>([\s\S]*?)<\/body\s*>/i;
const ARTIFACT_CSP = [
  "default-src 'none'",
  "base-uri 'none'",
  "form-action 'none'",
  "object-src 'none'",
  "frame-src 'none'",
  "child-src 'none'",
  "worker-src 'none'",
  "connect-src https://esm.sh",
  "manifest-src 'none'",
  "prefetch-src 'none'",
  "navigate-to 'none'",
  "img-src data: blob:",
  "media-src data: blob:",
  "font-src data:",
  "style-src 'unsafe-inline'",
  "script-src 'unsafe-inline' https://esm.sh",
].join("; ");

function parseFenceLanguage(info: string): string {
  const raw = info.trim().split(/\s+/)[0] ?? "";
  return raw.replace(/^\{?\.?/, "").replace(/\}?$/, "");
}

function artifactStableMessageID(
  message: Pick<ChatAreaMessage, "publicID" | "runID">,
): string {
  return message.runID?.trim() || message.publicID;
}

function isFenceClose(line: string, marker: string): boolean {
  const escaped = marker[0] === "`" ? "`" : "~";
  const re = new RegExp(`^[ \\t]*${escaped}{${marker.length},}[ \\t]*$`);
  return re.test(line);
}


function escapeHTML(value: string): string {
  return value
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;");
}

function escapeScriptContent(value: string): string {
  return value.replace(SCRIPT_CLOSE_RE, "<\\/script");
}

function escapeStyleContent(value: string): string {
  return value.replace(STYLE_CLOSE_RE, "<\\/style");
}

function scriptStringLiteral(value: string): string {
  return escapeScriptContent(JSON.stringify(value));
}

function artifactRuntimeScript(labels: Pick<ArtifactPreviewLabels, "unknownError">): string {
  return `<script>
(() => {
  const formatError = (value) => {
    if (!value) return ${scriptStringLiteral(labels.unknownError)};
    if (value && value.stack) return String(value.stack);
    if (value && value.message) return String(value.message);
    return String(value);
  };
  const showError = (value) => {
    const message = formatError(value);
    const node = document.createElement("pre");
    node.textContent = message;
    node.style.cssText = "margin:16px;padding:12px;border:1px solid CanvasText;border-radius:8px;background:Canvas;color:CanvasText;font:12px/1.5 ui-monospace,SFMono-Regular,Menlo,monospace;white-space:pre-wrap;";
    document.body.appendChild(node);
  };
  window.addEventListener("error", (event) => showError(event.error || event.message));
  window.addEventListener("unhandledrejection", (event) => showError(event.reason));
})();
</script>`;
}

function artifactPreviewResetStyle(): string {
  return `<style data-deeix-artifact-reset>
html,
body {
  min-height: 100%;
  width: 100%;
  margin: 0;
}

body {
  overflow: auto;
}

*,
*::before,
*::after {
  box-sizing: border-box;
}
</style>`;
}

function previewHead(title: string, labels: Pick<ArtifactPreviewLabels, "unknownError">): string {
  return [
    `<meta charset="utf-8">`,
    `<meta name="viewport" content="width=device-width, initial-scale=1">`,
    `<meta http-equiv="Content-Security-Policy" content="${ARTIFACT_CSP}">`,
    `<title>${escapeHTML(title)}</title>`,
    artifactPreviewResetStyle(),
    artifactRuntimeScript(labels),
  ].join("");
}

function htmlPreviewDocument(code: string, labels: ArtifactPreviewLabels): string {
  const safeHead = previewHead(labels.htmlTitle, labels);
  const userHead = HEAD_BLOCK_RE.exec(code)?.[1]?.trim() ?? "";
  const bodyMatch = BODY_BLOCK_RE.exec(code);
  const body = bodyMatch
    ? bodyMatch[1]
    : code
        .replace(DOCTYPE_RE, "")
        .replace(HTML_OPEN_RE, "")
        .replace(HTML_CLOSE_RE, "")
        .replace(HEAD_BLOCK_RE, "")
        .trim();

  return `<!doctype html><html><head>${safeHead}${userHead}</head><body>${body}</body></html>`;
}

function cssPreviewDocument(code: string, labels: ArtifactPreviewLabels): string {
  return `<!doctype html>
<html>
<head>
${previewHead(labels.cssTitle, labels)}
<style>${escapeStyleContent(code)}</style>
</head>
<body>
  <main class="artifact-preview">
    <section class="preview-panel">
      <p class="eyebrow">${escapeHTML(labels.cssEyebrow)}</p>
      <h1>${escapeHTML(labels.cssHeading)}</h1>
      <p>${escapeHTML(labels.cssDescription)}</p>
      <div class="preview-row">
        <button type="button">${escapeHTML(labels.cssPrimaryAction)}</button>
        <button type="button" class="secondary">${escapeHTML(labels.cssSecondaryAction)}</button>
      </div>
      <div class="preview-grid">
        <article><strong>${escapeHTML(labels.cssCardTitle)}</strong><span>${escapeHTML(labels.cssCardDescription)}</span></article>
        <article><strong>${escapeHTML(labels.cssMetricTitle)}</strong><span>128</span></article>
      </div>
    </section>
  </main>
</body>
</html>`;
}

function javascriptPreviewDocument(code: string, labels: ArtifactPreviewLabels): string {
  return `<!doctype html>
<html>
<head>
${previewHead(labels.jsTitle, labels)}
<style>
body { margin: 0; font: 14px/1.5 system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; color: CanvasText; background: Canvas; }
#root { min-height: 100vh; padding: 20px; box-sizing: border-box; }
.artifact-console { position: fixed; inset-inline: 12px; bottom: 12px; max-height: 32vh; overflow: auto; border: 1px solid ButtonBorder; border-radius: 8px; background: Canvas; color: CanvasText; padding: 10px; font: 12px/1.5 ui-monospace, SFMono-Regular, Menlo, monospace; white-space: pre-wrap; }
</style>
</head>
<body>
<div id="root"></div>
<pre id="console" class="artifact-console" hidden></pre>
<script>
(() => {
  const consoleNode = document.getElementById("console");
  const write = (level, values) => {
    consoleNode.hidden = false;
    consoleNode.textContent += "[" + level + "] " + values.map((item) => {
      try { return typeof item === "string" ? item : JSON.stringify(item); }
      catch { return String(item); }
    }).join(" ") + "\\n";
  };
  for (const level of ["log", "info", "warn", "error"]) {
    const original = console[level].bind(console);
    console[level] = (...values) => {
      write(level, values);
      original(...values);
    };
  }
})();
</script>
<script>${escapeScriptContent(code)}</script>
</body>
</html>`;
}

function svgPreviewDocument(code: string, labels: ArtifactPreviewLabels): string {
  return `<!doctype html>
<html>
<head>
${previewHead(labels.svgTitle, labels)}
<style>
body { min-height: 100vh; display: grid; place-items: center; padding: 20px; background: Canvas; }
svg { max-width: 100%; max-height: calc(100vh - 40px); }
</style>
</head>
<body>${code}</body>
</html>`;
}

function markdownToHTML(markdown: string): string {
  const lines = markdown.split(/\r?\n/);
  const html: string[] = [];
  let inList = false;
  let inCode = false;
  const closeList = () => {
    if (inList) {
      html.push("</ul>");
      inList = false;
    }
  };
  for (const line of lines) {
    if (/^\s*```/.test(line)) {
      if (inCode) {
        html.push("</code></pre>");
        inCode = false;
      } else {
        closeList();
        html.push("<pre><code>");
        inCode = true;
      }
      continue;
    }
    if (inCode) {
      html.push(`${escapeHTML(line)}\n`);
      continue;
    }
    const heading = /^(#{1,3})\s+(.+)$/.exec(line);
    if (heading) {
      closeList();
      const level = heading[1].length;
      html.push(`<h${level}>${inlineMarkdown(heading[2])}</h${level}>`);
      continue;
    }
    const listItem = /^\s*[-*]\s+(.+)$/.exec(line);
    if (listItem) {
      if (!inList) {
        html.push("<ul>");
        inList = true;
      }
      html.push(`<li>${inlineMarkdown(listItem[1])}</li>`);
      continue;
    }
    if (!line.trim()) {
      closeList();
      continue;
    }
    closeList();
    html.push(`<p>${inlineMarkdown(line)}</p>`);
  }
  closeList();
  if (inCode) {
    html.push("</code></pre>");
  }
  return html.join("\n");
}

function inlineMarkdown(value: string): string {
  return escapeHTML(value)
    .replace(/\*\*([^*]+)\*\*/g, "<strong>$1</strong>")
    .replace(/`([^`]+)`/g, "<code>$1</code>")
    .replace(/\[([^\]]+)\]\((https?:\/\/[^)\s]+)\)/g, '<a href="$2" target="_blank" rel="noreferrer">$1</a>');
}

function markdownPreviewDocument(code: string, labels: ArtifactPreviewLabels): string {
  return `<!doctype html>
<html>
<head>
${previewHead(labels.markdownTitle, labels)}
<style>
body { max-width: 760px; margin: 0 auto; padding: 24px; font: 15px/1.65 system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; color: CanvasText; background: Canvas; }
h1, h2, h3 { line-height: 1.2; margin: 1.2em 0 .55em; }
p, ul, pre { margin: 0 0 1em; }
pre { overflow: auto; padding: 12px; border-radius: 8px; background: Canvas; border: 1px solid ButtonBorder; }
code { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; }
a { color: LinkText; }
</style>
</head>
<body>${markdownToHTML(code)}</body>
</html>`;
}

function mermaidPreviewDocument(code: string, labels: ArtifactPreviewLabels): string {
  return `<!doctype html>
<html>
<head>
${previewHead(labels.mermaidTitle, labels)}
<style>
body { min-height: 100vh; margin: 0; display: grid; place-items: center; padding: 20px; background: Canvas; color: CanvasText; }
#mermaid-root { width: min(100%, 960px); overflow: auto; }
</style>
</head>
<body>
<div id="mermaid-root" class="mermaid">${escapeHTML(code)}</div>
<script type="module">
import mermaid from "https://esm.sh/mermaid@11";
mermaid.initialize({ startOnLoad: true, securityLevel: "strict" });
</script>
</body>
</html>`;
}

function reactPreviewDocument(code: string, labels: ArtifactPreviewLabels): string {
  return `<!doctype html>
<html>
<head>
${previewHead(labels.reactTitle, labels)}
<style>
body { margin: 0; font: 14px/1.5 system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; color: CanvasText; background: Canvas; }
#root { min-height: 100vh; padding: 20px; box-sizing: border-box; }
</style>
</head>
<body>
<div id="root"></div>
<script type="module">
import React from "https://esm.sh/react@19";
import { createRoot } from "https://esm.sh/react-dom@19/client";
import Babel from "https://esm.sh/@babel/standalone@7";
const source = ${JSON.stringify(code)};
const compiled = Babel.transform(source, { presets: [["env", { modules: "commonjs" }], "react", "typescript"], filename: "artifact.tsx" }).code;
const module = { exports: {} };
const exports = module.exports;
const require = (name) => {
  if (name === "react") return React;
  throw new Error(${scriptStringLiteral(labels.reactUnsupportedImport)} + " " + name);
};
const App = new Function("React", "module", "exports", "require", compiled + "\\nreturn module.exports.default || exports.default || (typeof App !== 'undefined' ? App : null);")(React, module, exports, require);
if (!App) throw new Error(${scriptStringLiteral(labels.reactMissingComponent)});
createRoot(document.getElementById("root")).render(React.createElement(App));
</script>
</body>
</html>`;
}

export function buildArtifactPreviewDocument(kind: ArtifactPreviewKind, code: string, labels: ArtifactPreviewLabels): string {
  if (kind === "css") return cssPreviewDocument(code, labels);
  if (kind === "javascript") return javascriptPreviewDocument(code, labels);
  if (kind === "svg") return svgPreviewDocument(code, labels);
  if (kind === "mermaid") return mermaidPreviewDocument(code, labels);
  if (kind === "markdown") return markdownPreviewDocument(code, labels);
  if (kind === "react") return reactPreviewDocument(code, labels);
  return htmlPreviewDocument(code, labels);
}

export function resolveArtifactDownloadName(kind: ArtifactPreviewKind): string {
  if (kind === "css") return "artifact-css-preview.html";
  if (kind === "javascript") return "artifact-js-preview.html";
  if (kind === "svg") return "artifact-svg-preview.html";
  if (kind === "mermaid") return "artifact-mermaid-preview.html";
  if (kind === "markdown") return "artifact-markdown-preview.html";
  if (kind === "react") return "artifact-react-preview.html";
  return "artifact-preview.html";
}

export function downloadArtifactHTML(fileName: string, value: string): void {
  const blob = new Blob([value], { type: "text/html;charset=utf-8" });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = fileName;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(url);
}

export function extractArtifactsFromContent(
  message: Pick<ChatAreaMessage, "content" | "isStreaming" | "key" | "publicID" | "runID" | "updatedAt">,
): ChatArtifact[] {
  const content = message.content;
  const artifacts: ChatArtifact[] = [];
  const lines = content.split(/\r?\n/);
  const stableMessageID = artifactStableMessageID(message);
  const runID = message.runID?.trim() || undefined;
  let openMarker = "";
  let language = "";
  let codeLines: string[] = [];
  let blockIndex = 0;

  const pushArtifact = (code: string, complete: boolean) => {
    const kind = resolveArtifactPreviewKind(language, code);
    if (!kind || !code.trim()) {
      return;
    }
    artifacts.push({
      id: `${stableMessageID}:artifact:${blockIndex}`,
      messageID: message.publicID,
      messageKey: message.key,
      runID,
      blockIndex,
      kind,
      language,
      code,
      complete,
      streaming: Boolean(message.isStreaming),
      updatedAt: message.updatedAt,
    });
    blockIndex += 1;
  };

  for (const line of lines) {
    if (!openMarker) {
      const openMatch = line.match(FENCE_OPEN_RE);
      if (!openMatch) {
        continue;
      }
      openMarker = openMatch[1] ?? "";
      language = parseFenceLanguage(openMatch[2] ?? "");
      codeLines = [];
      continue;
    }

    if (isFenceClose(line, openMarker)) {
      pushArtifact(codeLines.join("\n"), true);
      openMarker = "";
      language = "";
      codeLines = [];
      continue;
    }

    codeLines.push(line);
  }

  if (openMarker && message.isStreaming) {
    pushArtifact(codeLines.join("\n"), false);
  }

  if (artifacts.length === 0) {
    const kind = resolveArtifactPreviewKind("", content);
    if (kind && content.trim()) {
      artifacts.push({
        id: `${stableMessageID}:artifact:0`,
        messageID: message.publicID,
        messageKey: message.key,
        runID,
        blockIndex: 0,
        kind,
        language: kind,
        code: content,
        complete: !message.isStreaming,
        streaming: Boolean(message.isStreaming),
        updatedAt: message.updatedAt,
      });
    }
  }

  return artifacts;
}

export function extractArtifactsFromMessages(messages: ChatAreaMessage[]): ChatArtifact[] {
  return messages.flatMap((message) => (message.role === "assistant" ? extractArtifactsFromContent(message) : []));
}
