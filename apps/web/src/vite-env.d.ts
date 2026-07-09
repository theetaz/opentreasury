/// <reference types="vite/client" />

interface ImportMetaEnv {
  /** Base URL of the core API; unset means simulated data mode. */
  readonly VITE_CORE_API_URL?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
