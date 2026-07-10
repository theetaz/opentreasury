// Design tokens mirrored from docs/design/tokens.css. The app ships
// dark-only in v1 (the design language is dark-first, and react-native-web's
// appearance detection is inconsistent across navigation chrome); the light
// palette stays here for the follow-up.

export interface Palette {
  background: string;
  surface: string;
  surfaceRaised: string;
  border: string;
  text: string;
  textMuted: string;
  primary: string;
  success: string;
  warning: string;
  danger: string;
}

export const palettes: { dark: Palette; light: Palette } = {
  dark: {
    background: "#0a0a0c",
    surface: "#131316",
    surfaceRaised: "#1a1a1f",
    border: "rgba(255, 255, 255, 0.08)",
    text: "#f4f4f5",
    textMuted: "#a1a1aa",
    primary: "#14b8a6",
    success: "#34c759",
    warning: "#f0a820",
    danger: "#f2555a",
  },
  light: {
    background: "#f7f7f8",
    surface: "#ffffff",
    surfaceRaised: "#ffffff",
    border: "rgba(9, 9, 11, 0.08)",
    text: "#0b0b0e",
    textMuted: "#52525b",
    primary: "#0d9488",
    success: "#15803d",
    warning: "#b45309",
    danger: "#b91c1c",
  },
};

export function usePalette(): Palette {
  return palettes.dark;
}

export const spacing = { xs: 4, sm: 8, md: 12, lg: 16, xl: 24 } as const;
export const radius = { card: 12, control: 8, pill: 999 } as const;
