export type Meta = {
  name?: string;
  version?: string;
  demo: boolean;
  demo_lock: boolean;
  single_tenant?: boolean;
  default_org?: string;
};

export type Agent = { id: string; name: string; created_at: string };
export type Tool = {
  id: string;
  name: string;
  kind: string;
  config: unknown;
  seeded: boolean;
  created_at: string;
};
export type Policy = {
  id: string;
  version: number;
  document: unknown;
  active: boolean;
  created_at: string;
};
export type Approval = {
  id: string;
  status: string;
  comment: string;
  decided_by: string;
  expires_at: string;
  agent_id: string;
  tool_id: string;
  tool_name: string;
  args_redacted: unknown;
  args_hash: string;
  rule_id: string;
  reason: string;
  request_id: string;
  decision: string;
};
export type AuditEvent = {
  seq: number;
  event_type: string;
  payload: unknown;
  created_at: string;
};
export type Verify = {
  ok: boolean;
  events: number;
  head?: string;
  broken_seq?: number;
  reason?: string;
};

export class KwError extends Error {}

function baseURL() {
  const base = process.env.KEYWARDEN_API_URL?.replace(/\/$/, "");
  if (!base) throw new KwError("KEYWARDEN_API_URL is not set");
  return base;
}

export async function kw<T>(path: string, init?: RequestInit): Promise<T> {
  const token = process.env.KEYWARDEN_ADMIN_TOKEN;
  if (!token) throw new KwError("KEYWARDEN_ADMIN_TOKEN is not set");
  const res = await fetch(baseURL() + path, {
    ...init,
    cache: "no-store",
    headers: {
      Authorization: `Bearer ${token}`,
      ...(init?.body ? { "Content-Type": "application/json" } : {}),
      ...init?.headers,
    },
  });
  const text = await res.text();
  if (!res.ok) {
    let message = text;
    try {
      const parsed = JSON.parse(text) as { error?: string };
      if (parsed.error) message = parsed.error;
    } catch {
      /* keep the raw body */
    }
    throw new KwError(message || res.statusText);
  }
  if (!text) return undefined as T;
  return JSON.parse(text) as T;
}

export async function loadMeta(): Promise<Meta> {
  const base = process.env.KEYWARDEN_API_URL?.replace(/\/$/, "");
  if (!base) return { demo: false, demo_lock: false };
  try {
    const res = await fetch(base + "/v1/meta", { cache: "no-store" });
    if (!res.ok) return { demo: false, demo_lock: false };
    const body = (await res.json()) as Meta;
    return { demo: Boolean(body.demo), demo_lock: Boolean(body.demo_lock), version: body.version, name: body.name };
  } catch {
    return { demo: false, demo_lock: false };
  }
}

export function when(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toISOString().replace("T", " ").slice(0, 16) + " UTC";
}

export function pretty(value: unknown) {
  return JSON.stringify(value, null, 2);
}
