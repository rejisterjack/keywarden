/** Use Secure cookies only on HTTPS deployments (or when explicitly forced). */
export function secureCookies(): boolean {
  const force = process.env.KEYWARDEN_COOKIE_SECURE?.trim().toLowerCase();
  if (force === "1" || force === "true" || force === "yes") return true;
  if (force === "0" || force === "false" || force === "no") return false;
  const api = (process.env.KEYWARDEN_API_URL ?? "").trim();
  return api.startsWith("https://");
}