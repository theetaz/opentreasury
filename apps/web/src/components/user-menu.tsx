import { LogOut, UserRound } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger
} from "@/components/ui/dropdown-menu";
import { authEnabled, useAuthStore } from "@/stores/auth";

export function UserMenu() {
  const user = useAuthStore((state) => state.user);
  const logout = useAuthStore((state) => state.logout);
  const { t } = useTranslation();

  if (!authEnabled || !user) {
    return null;
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="sm" className="gap-2">
          <UserRound aria-hidden className="size-4" />
          <span className="max-w-32 truncate">{user.username}</span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-56">
        <DropdownMenuLabel className="space-y-1">
          <div className="font-medium">{user.username}</div>
          <div className="text-xs font-normal text-muted-foreground">
            {user.roles.join(", ") || "no roles"}
          </div>
          {user.institutionId ? (
            <div className="text-xs font-normal text-muted-foreground">
              Institution: <span className="font-mono">{user.institutionId}</span>
            </div>
          ) : null}
        </DropdownMenuLabel>
        <DropdownMenuSeparator />
        <DropdownMenuItem onClick={logout}>
          <LogOut aria-hidden />
          {t("auth.signOut")}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
