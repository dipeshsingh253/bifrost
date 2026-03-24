export const APPEARANCE_STORAGE_KEY = "bifrost:appearance";
export const APPEARANCE_CHANGE_EVENT = "bifrost:appearance-change";

export type ThemePreference = "system" | "dark" | "light";
export type DefaultTimeRangePreference = "1h" | "6h" | "24h" | "7d";

export type AppearanceSettings = {
  theme: ThemePreference;
  defaultTimeRange: DefaultTimeRangePreference;
};

export const defaultAppearanceSettings: AppearanceSettings = {
  theme: "system",
  defaultTimeRange: "1h",
};

function isThemePreference(value: unknown): value is ThemePreference {
  return value === "system" || value === "dark" || value === "light";
}

function isDefaultTimeRangePreference(value: unknown): value is DefaultTimeRangePreference {
  return value === "1h" || value === "6h" || value === "24h" || value === "7d";
}

export function normalizeAppearanceSettings(value: unknown): AppearanceSettings {
  if (!value || typeof value !== "object") {
    return defaultAppearanceSettings;
  }

  const candidate = value as Partial<AppearanceSettings>;

  return {
    theme: isThemePreference(candidate.theme) ? candidate.theme : defaultAppearanceSettings.theme,
    defaultTimeRange: isDefaultTimeRangePreference(candidate.defaultTimeRange)
      ? candidate.defaultTimeRange
      : defaultAppearanceSettings.defaultTimeRange,
  };
}

export function readAppearanceSettings(): AppearanceSettings {
  if (typeof window === "undefined") {
    return defaultAppearanceSettings;
  }

  try {
    const rawValue = window.localStorage.getItem(APPEARANCE_STORAGE_KEY);
    if (!rawValue) {
      return defaultAppearanceSettings;
    }

    return normalizeAppearanceSettings(JSON.parse(rawValue));
  } catch {
    return defaultAppearanceSettings;
  }
}

export function resolveTheme(theme: ThemePreference): "dark" | "light" {
  if (theme === "dark" || theme === "light") {
    return theme;
  }

  if (typeof window !== "undefined" && window.matchMedia("(prefers-color-scheme: dark)").matches) {
    return "dark";
  }

  return "light";
}

export function applyTheme(theme: ThemePreference) {
  if (typeof document === "undefined") {
    return;
  }

  const root = document.documentElement;
  const resolvedTheme = resolveTheme(theme);
  root.classList.toggle("dark", resolvedTheme === "dark");
  root.classList.toggle("light", resolvedTheme === "light");
}

export function saveAppearanceSettings(settings: AppearanceSettings) {
  if (typeof window === "undefined") {
    return;
  }

  window.localStorage.setItem(APPEARANCE_STORAGE_KEY, JSON.stringify(settings));
  applyTheme(settings.theme);
  window.dispatchEvent(new CustomEvent(APPEARANCE_CHANGE_EVENT, { detail: settings }));
}
