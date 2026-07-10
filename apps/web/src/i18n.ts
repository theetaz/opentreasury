// App internationalization: English is the source language, and every locale
// ships as a complete JSON file under src/locales/. The choice persists in
// localStorage and is reflected on <html lang> for assistive technology.
import i18next from "i18next";
import { initReactI18next } from "react-i18next";

import en from "./locales/en.json";
import sw from "./locales/sw.json";

const STORAGE_KEY = "opentreasury-locale";

export const locales = [
  { code: "en", label: "English" },
  { code: "sw", label: "Kiswahili" },
] as const;

export type LocaleCode = (typeof locales)[number]["code"];

const stored =
  typeof window !== "undefined" ? window.localStorage.getItem(STORAGE_KEY) : null;

void i18next.use(initReactI18next).init({
  resources: { en: { translation: en }, sw: { translation: sw } },
  lng: stored && locales.some((locale) => locale.code === stored) ? stored : "en",
  fallbackLng: "en",
  interpolation: { escapeValue: false },
});

function applyLanguage(language: string) {
  if (typeof document !== "undefined") {
    document.documentElement.lang = language;
  }
}

applyLanguage(i18next.language);
i18next.on("languageChanged", (language) => {
  applyLanguage(language);
  if (typeof window !== "undefined") {
    window.localStorage.setItem(STORAGE_KEY, language);
  }
});

export const i18n = i18next;
