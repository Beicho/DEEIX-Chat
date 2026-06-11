export function formatSelectedQuote(value: string) {
  const normalized = value.replace(/\r\n?/g, "\n").trim();
  if (!normalized) {
    return "";
  }
  return normalized
    .split("\n")
    .map((line) => (line.trim() ? `> ${line.trimEnd()}` : ">"))
    .join("\n");
}

export function buildQuotedDraft(currentDraft: string, selectedText: string) {
  const quote = formatSelectedQuote(selectedText);
  if (!quote) {
    return currentDraft;
  }

  const existingDraft = currentDraft.trim();
  return existingDraft ? `${quote}\n\n${existingDraft}` : `${quote}\n\n`;
}
