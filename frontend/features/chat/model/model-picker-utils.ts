export const MODEL_PICKER_RECENT_STORAGE_KEY = "deeix-chat:recent-models:v1";
export const MODEL_PICKER_RECENT_LIMIT = 5;

export type ModelPickerSearchableItem = {
  platformModelName: string;
  vendor?: string | null;
  icon?: string | null;
};

export type ModelPickerGroup<TItem extends ModelPickerSearchableItem> = {
  vendor: string;
  label: string;
  icon: string;
  items: TItem[];
};

function normalizeSearchValue(value: string): string {
  return value.trim().toLocaleLowerCase();
}

export function filterModelPickerGroups<TItem extends ModelPickerSearchableItem>(
  groups: readonly ModelPickerGroup<TItem>[],
  query: string,
): ModelPickerGroup<TItem>[] {
  const normalizedQuery = normalizeSearchValue(query);
  if (!normalizedQuery) {
    return groups.map((group) => ({ ...group, items: [...group.items] }));
  }

  return groups.reduce<ModelPickerGroup<TItem>[]>((result, group) => {
    const normalizedVendor = normalizeSearchValue(`${group.label} ${group.vendor} ${group.icon}`);
    const items = group.items.filter((item) => {
      const searchableText = normalizeSearchValue(
        `${item.platformModelName} ${item.vendor ?? ""} ${item.icon ?? ""} ${group.label} ${group.vendor}`,
      );
      return searchableText.includes(normalizedQuery) || normalizedVendor.includes(normalizedQuery);
    });
    if (items.length > 0) {
      result.push({ ...group, items });
    }
    return result;
  }, []);
}

export function recordRecentModelSelection(
  currentNames: readonly string[],
  selectedPlatformModelName: string,
  limit = MODEL_PICKER_RECENT_LIMIT,
): string[] {
  const selected = selectedPlatformModelName.trim();
  if (!selected) {
    return currentNames.slice(0, limit);
  }

  const seen = new Set<string>();
  const next = [selected, ...currentNames].reduce<string[]>((result, name) => {
    const normalizedName = name.trim();
    if (!normalizedName || seen.has(normalizedName)) {
      return result;
    }
    seen.add(normalizedName);
    result.push(normalizedName);
    return result;
  }, []);

  return next.slice(0, Math.max(1, limit));
}

export function resolveRecentModelOptions<TItem extends ModelPickerSearchableItem>(
  modelOptions: readonly TItem[],
  recentNames: readonly string[],
): TItem[] {
  const byName = new Map(modelOptions.map((item) => [item.platformModelName, item]));
  return recentNames.reduce<TItem[]>((result, name) => {
    const item = byName.get(name);
    if (item) {
      result.push(item);
    }
    return result;
  }, []);
}

export function readRecentModelNames(
  storage: Pick<Storage, "getItem">,
  key = MODEL_PICKER_RECENT_STORAGE_KEY,
): string[] {
  try {
    const rawValue = storage.getItem(key);
    if (!rawValue) {
      return [];
    }
    const parsedValue: unknown = JSON.parse(rawValue);
    if (!Array.isArray(parsedValue)) {
      return [];
    }
    return parsedValue.filter((item): item is string => typeof item === "string" && item.trim().length > 0);
  } catch {
    return [];
  }
}

export function writeRecentModelNames(
  storage: Pick<Storage, "setItem">,
  names: readonly string[],
  key = MODEL_PICKER_RECENT_STORAGE_KEY,
): void {
  try {
    storage.setItem(key, JSON.stringify([...names]));
  } catch {
    // Storage can be unavailable in hardened browsing modes.
  }
}
