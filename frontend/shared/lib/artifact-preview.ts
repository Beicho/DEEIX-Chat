export type ArtifactPreviewKind = "html" | "css" | "javascript" | "svg" | "mermaid" | "markdown" | "react";

const HTML_LIKE_RE = /^\s*(?:<!doctype\s+html|<html\b|<head\b|<body\b|<(?:article|canvas|div|main|section|style|script|svg)\b)/i;

function normalizeLanguage(language: string): string {
  return language.trim().toLowerCase();
}

export function resolveArtifactPreviewKind(language: string, code: string): ArtifactPreviewKind | null {
  const normalized = normalizeLanguage(language);
  if (["html", "htm", "xhtml"].includes(normalized)) return "html";
  if (["css", "scss", "sass", "less"].includes(normalized)) return "css";
  if (["js", "javascript", "mjs", "cjs"].includes(normalized)) return "javascript";
  if (["svg"].includes(normalized)) return "svg";
  if (["mermaid", "mmd"].includes(normalized)) return "mermaid";
  if (["md", "markdown"].includes(normalized)) return "markdown";
  if (["jsx", "tsx", "react"].includes(normalized)) return "react";
  if ((!normalized || normalized === "markdown") && HTML_LIKE_RE.test(code)) return "html";
  return null;
}
