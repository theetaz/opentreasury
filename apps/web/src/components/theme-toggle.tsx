import { Moon, Sun } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { useThemeStore } from "@/stores/theme";

export function ThemeToggle() {
  const theme = useThemeStore((state) => state.theme);
  const toggle = useThemeStore((state) => state.toggle);
  const { t } = useTranslation();

  return (
    <Button variant="ghost" size="icon" aria-label={t("shell.toggleTheme")} onClick={toggle}>
      {theme === "dark" ? <Sun aria-hidden /> : <Moon aria-hidden />}
    </Button>
  );
}
