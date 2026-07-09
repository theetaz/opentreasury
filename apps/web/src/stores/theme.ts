import { create } from "zustand";
import { persist } from "zustand/middleware";

export type Theme = "dark" | "light";

export function applyTheme(theme: Theme) {
  document.documentElement.classList.toggle("dark", theme === "dark");
}

type ThemeState = {
  theme: Theme;
  setTheme: (theme: Theme) => void;
  toggle: () => void;
};

export const useThemeStore = create<ThemeState>()(
  persist(
    (set, get) => ({
      // Operator dashboards are dark-first (design language §1.3).
      theme: "dark",
      setTheme: (theme) => {
        applyTheme(theme);
        set({ theme });
      },
      toggle: () => get().setTheme(get().theme === "dark" ? "light" : "dark")
    }),
    {
      name: "opentreasury-theme",
      onRehydrateStorage: () => (state) => {
        if (state) applyTheme(state.theme);
      }
    }
  )
);
