import { createHash, createHmac, timingSafeEqual } from "crypto";
import { cookies } from "next/headers";
import { redirect } from "next/navigation";

const cookieName = "kw_session";
const maxAge = 60 * 60 * 12;

function secret() {
  const value = process.env.KEYWARDEN_SESSION_SECRET;
  if (!value || value.length < 16) {
    throw new Error("KEYWARDEN_SESSION_SECRET must be at least 16 characters");
  }
  return value;
}

function sign(exp: number) {
  const body = Buffer.from(JSON.stringify({ exp })).toString("base64url");
  const mac = createHmac("sha256", secret()).update(body).digest("base64url");
  return `${body}.${mac}`;
}

export function passwordMatches(password: string) {
  const want = process.env.KEYWARDEN_DASHBOARD_PASSWORD ?? "";
  if (want.length < 8) return false;
  const a = createHash("sha256").update(password).digest();
  const b = createHash("sha256").update(want).digest();
  return timingSafeEqual(a, b);
}

export async function hasSession() {
  const jar = await cookies();
  const raw = jar.get(cookieName)?.value;
  if (!raw) return false;
  const [body, mac] = raw.split(".");
  if (!body || !mac) return false;
  const expect = createHmac("sha256", secret()).update(body).digest("base64url");
  const left = Buffer.from(mac);
  const right = Buffer.from(expect);
  if (left.length !== right.length || !timingSafeEqual(left, right)) return false;
  try {
    const payload = JSON.parse(Buffer.from(body, "base64url").toString()) as { exp?: number };
    return typeof payload.exp === "number" && payload.exp > Date.now() / 1000;
  } catch {
    return false;
  }
}

export async function requireSession() {
  if (!(await hasSession())) redirect("/login");
}

export async function startSession() {
  const jar = await cookies();
  jar.set(cookieName, sign(Math.floor(Date.now() / 1000) + maxAge), {
    httpOnly: true,
    sameSite: "lax",
    secure: process.env.NODE_ENV === "production",
    path: "/",
    maxAge,
  });
}

export async function clearSession() {
  const jar = await cookies();
  jar.delete(cookieName);
}
