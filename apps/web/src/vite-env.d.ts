/// <reference types="vite/client" />

interface ImportMetaEnv {
  /** Base URL of the core API; unset means simulated data mode. */
  readonly VITE_CORE_API_URL?: string;
  /** OIDC issuer/authority URL; unset means auth is disabled (demo mode). */
  readonly VITE_OIDC_AUTHORITY?: string;
  /** OIDC client id; defaults to opentreasury-web. */
  readonly VITE_OIDC_CLIENT_ID?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
