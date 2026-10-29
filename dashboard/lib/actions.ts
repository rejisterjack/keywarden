"use server";

import { revalidatePath } from "next/cache";
import { redirect } from "next/navigation";
import { kw, KwError, loadMeta } from "@/lib/api";
import { clearSession, passwordMatches, requireSession, startSession } from "@/lib/session";

function fail(path: string, err: unknown): never {
  const message = err instanceof KwError ? err.message : "request failed";
  redirect(`${path}?error=${encodeURIComponent(message)}`);
}

export async function login(formData: FormData) {
  const password = String(formData.get("password") ?? "");
  if (!passwordMatches(password)) {
    redirect("/login?error=" + encodeURIComponent("Wrong password"));
  }
  await startSession();
  redirect("/");
}

export async function logout() {
  await clearSession();
  redirect("/login");
}

export async function decide(formData: FormData) {
  await requireSession();
  const id = String(formData.get("id") ?? "");
  const decision = String(formData.get("decision") ?? "");
  const comment = String(formData.get("comment") ?? "");
  try {
    await kw(`/v1/approvals/${id}`, {
      method: "POST",
      body: JSON.stringify({ decision, comment }),
    });
  } catch (err) {
    fail("/", err);
  }
  revalidatePath("/");
}

export async function createAgent(formData: FormData) {
  await requireSession();
  await rejectIfLocked("/agents");
  try {
    await kw("/v1/agents", {
      method: "POST",
      body: JSON.stringify({
        id: String(formData.get("id") ?? "").trim(),
        name: String(formData.get("name") ?? "").trim(),
      }),
    });
  } catch (err) {
    fail("/agents", err);
  }
  revalidatePath("/agents");
}

export async function deleteAgent(formData: FormData) {
  await requireSession();
  await rejectIfLocked("/agents");
  const id = String(formData.get("id") ?? "");
  try {
    await kw(`/v1/agents/${id}`, { method: "DELETE" });
  } catch (err) {
    fail("/agents", err);
  }
  revalidatePath("/agents");
}

export async function createTool(formData: FormData) {
  await requireSession();
  await rejectIfLocked("/tools");
  let config: unknown;
  let secret: unknown;
  try {
    config = JSON.parse(String(formData.get("config") ?? "{}"));
    secret = JSON.parse(String(formData.get("secret") ?? ""));
  } catch {
    fail("/tools", new KwError("config and secret must be JSON"));
  }
  try {
    await kw("/v1/tools", {
      method: "POST",
      body: JSON.stringify({
        id: String(formData.get("id") ?? "").trim(),
        name: String(formData.get("name") ?? "").trim(),
        kind: String(formData.get("kind") ?? ""),
        config,
        secret,
      }),
    });
  } catch (err) {
    fail("/tools", err);
  }
  revalidatePath("/tools");
}

export async function deleteTool(formData: FormData) {
  await requireSession();
  await rejectIfLocked("/tools");
  const id = String(formData.get("id") ?? "");
  try {
    await kw(`/v1/tools/${id}`, { method: "DELETE" });
  } catch (err) {
    fail("/tools", err);
  }
  revalidatePath("/tools");
}

export async function applyPolicy(formData: FormData) {
  await requireSession();
  await rejectIfLocked("/policy");
  const raw = String(formData.get("document") ?? "");
  try {
    JSON.parse(raw);
  } catch {
    fail("/policy", new KwError("policy must be JSON"));
  }
  try {
    await kw("/v1/policies", { method: "POST", body: raw });
  } catch (err) {
    fail("/policy", err);
  }
  revalidatePath("/policy");
}

async function rejectIfLocked(path: string) {
  const meta = await loadMeta();
  if (meta.demo_lock) {
    fail(path, new KwError("the hosted demo does not accept this change"));
  }
}
