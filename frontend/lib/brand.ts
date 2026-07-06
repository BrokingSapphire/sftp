// Typed accessor for the white-label branding config.
//
// The canonical file is /brand.config.json at the repository root — edit that.
// `npm run predev` / `prebuild` (and the Docker build) sync it into
// config/brand.json, which is imported here so the value is available at build
// time everywhere (server + client).
import brand from "@/config/brand.json";

export interface Brand {
  company: {
    name: string;
    shortName: string;
    product: string;
    productShort: string;
    tagline: string;
    description: string;
    url: string;
    copyright: string;
  };
  logo: { full: string; light: string; dark: string; favicon: string };
  colors: { primary: string; primaryForeground: string; primaryDark: string; primaryForegroundDark: string };
  org: { domains: string[]; supportEmail: string };
  mail: { from: string };
  // Only microsoft.enabled reaches the browser — credentials are backend-only.
  sso?: { microsoft?: { enabled: boolean } };
}

export const BRAND = brand as Brand;

/** Org email domains (lower-cased) used to flag external shares. */
export const ORG_DOMAINS = BRAND.org.domains.map((d) => d.trim().toLowerCase()).filter(Boolean);

export function isExternalEmail(email: string): boolean {
  const at = email.lastIndexOf("@");
  if (at < 0) return false;
  const domain = email.slice(at + 1).toLowerCase();
  return ORG_DOMAINS.length > 0 && !ORG_DOMAINS.includes(domain);
}
