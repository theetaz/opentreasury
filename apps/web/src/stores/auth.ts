import { create } from "zustand";
import { User, UserManager, WebStorageStateStore } from "oidc-client-ts";

const authority = import.meta.env.VITE_OIDC_AUTHORITY;
const clientId = import.meta.env.VITE_OIDC_CLIENT_ID ?? "opentreasury-web";

// Auth is enabled only when an OIDC authority is configured; otherwise the app
// runs open (demo mode) exactly as before.
export const authEnabled = Boolean(authority);

const userManager = authEnabled
  ? new UserManager({
      authority: authority as string,
      client_id: clientId,
      redirect_uri: `${window.location.origin}/auth/callback`,
      post_logout_redirect_uri: window.location.origin,
      response_type: "code",
      scope: "openid profile",
      userStore: new WebStorageStateStore({ store: window.localStorage })
    })
  : null;

export type AuthUser = {
  username: string;
  roles: string[];
  institutionId?: string;
  accessToken: string;
};

type AuthState = {
  user: AuthUser | null;
  ready: boolean;
  login: () => void;
  logout: () => void;
  handleCallback: () => Promise<void>;
  loadUser: () => Promise<void>;
};

function toAuthUser(user: User): AuthUser {
  const profile = user.profile as {
    preferred_username?: string;
    institution_id?: string;
    realm_access?: { roles?: string[] };
  };
  return {
    username: profile.preferred_username ?? "user",
    roles: profile.realm_access?.roles ?? [],
    institutionId: profile.institution_id,
    accessToken: user.access_token
  };
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  ready: !authEnabled,
  login: () => {
    // Carry the deep link through the OIDC round trip so a bookmarked page
    // lands where the user intended after signing in.
    const returnTo = window.location.pathname + window.location.search;
    void userManager?.signinRedirect({ state: { returnTo } });
  },
  logout: () => {
    set({ user: null });
    void userManager?.signoutRedirect();
  },
  handleCallback: async () => {
    if (!userManager) return;
    let returnTo = "/";
    try {
      const user = await userManager.signinRedirectCallback();
      const state = user.state as { returnTo?: string } | undefined;
      // Only same-origin paths: anything else would be an open redirect.
      if (state?.returnTo?.startsWith("/") && !state.returnTo.startsWith("//")) {
        returnTo = state.returnTo;
      }
      set({ user: toAuthUser(user), ready: true });
    } finally {
      // Full navigation (not history.replaceState): the router must
      // re-initialize at the destination — replaceState is invisible to it
      // and leaves the app rendering the unrouted /auth/callback path.
      window.location.replace(returnTo);
    }
  },
  loadUser: async () => {
    if (!userManager) {
      set({ ready: true });
      return;
    }
    const user = await userManager.getUser();
    set({ user: user && !user.expired ? toAuthUser(user) : null, ready: true });
  }
}));

/** Current access token for API calls, or null when unauthenticated/demo. */
export function currentAccessToken(): string | null {
  return useAuthStore.getState().user?.accessToken ?? null;
}
