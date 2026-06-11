export type PromptTemplate = {
  id: string;
  title: string;
  body: string;
  category: string;
};

export function normalizePromptTemplate(value: PromptTemplate): PromptTemplate {
  return {
    id: value.id.trim(),
    title: value.title.trim(),
    body: value.body.trim(),
    category: value.category.trim(),
  };
}

export function filterPromptTemplates(templates: PromptTemplate[], query: string) {
  const normalizedQuery = query.trimStart().replace(/^\/+/, "").trim().toLowerCase();
  if (!normalizedQuery) {
    return templates;
  }
  return templates.filter((template) => {
    const haystack = `${template.title} ${template.body} ${template.category}`.toLowerCase();
    return haystack.includes(normalizedQuery);
  });
}

export function buildDraftFromPromptTemplate(currentDraft: string, template: PromptTemplate) {
  const commandDraft = currentDraft.trimStart();
  const slashCommandMatch = commandDraft.match(/^\/\S+\s*/);
  const draftTail = slashCommandMatch ? commandDraft.slice(slashCommandMatch[0].length).trim() : currentDraft.trim();
  return draftTail ? `${template.body}\n\n${draftTail}` : template.body;
}

export function recordPromptTemplateUsage(existingIDs: string[], templateID: string, limit = 5) {
  const normalizedID = templateID.trim();
  if (!normalizedID) {
    return existingIDs.slice(0, limit);
  }
  return [normalizedID, ...existingIDs.filter((id) => id !== normalizedID)].slice(0, limit);
}
