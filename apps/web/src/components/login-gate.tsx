import { useEffect } from "react";
import { ShieldCheck } from "lucide-react";
import { Button } from "@/components/ui/button";
import { authEnabled, useAuthStore } from "@/stores/auth";

// Gates the app behind OIDC login when authentication is enabled. In demo mode
// (no OIDC authority configured) it renders children immediately.
export function LoginGate({ children }: { children: React.ReactNode }) {
  const user = useAuthStore((state) => state.user);
  const ready = useAuthStore((state) => state.ready);
  const login = useAuthStore((state) => state.login);
  const loadUser = useAuthStore((state) => state.loadUser);
  const handleCallback = useAuthStore((state) => state.handleCallback);

  useEffect(() => {
    if (window.location.pathname === "/auth/callback") {
      void handleCallback();
    } else {
      void loadUser();
    }
  }, [handleCallback, loadUser]);

  if (!authEnabled || user) {
    return <>{children}</>;
  }

  if (!ready || window.location.pathname === "/auth/callback") {
    return (
      <div className="grid min-h-svh place-items-center text-sm text-muted-foreground">
        Signing in…
      </div>
    );
  }

  return (
    <div className="grid min-h-svh place-items-center bg-background p-6">
      <div className="flex w-full max-w-sm flex-col items-center gap-6 rounded-xl border bg-card p-8 text-center shadow-sm">
        <div className="grid size-12 place-items-center rounded-xl bg-primary text-primary-foreground">
          <ShieldCheck aria-hidden />
        </div>
        <div>
          <h1 className="text-lg font-semibold">OpenTreasury</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Sign in to access treasury data.
          </p>
        </div>
        <Button className="w-full" onClick={login}>
          Sign in
        </Button>
        <p className="text-xs text-muted-foreground">
          Demo users: treasury-admin · institution-user · auditor
        </p>
      </div>
    </div>
  );
}
