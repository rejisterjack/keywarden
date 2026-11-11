import { expect, test } from "@playwright/test";

const password = "local-demo-password";

test("login, task token, and audit chain", async ({ page }) => {
  await page.goto("/login");
  await page.locator("#password").fill("not-the-password");
  await page.getByRole("button", { name: "Sign In to Operator Console" }).click();
  await expect(page).toHaveURL(/\/login/);
  await expect(page.getByText("Wrong password")).toBeVisible();

  await page.locator("#password").fill(password);
  await page.getByRole("button", { name: "Sign In to Operator Console" }).click();
  await expect(page).toHaveURL(/\/approvals/);

  const session = (await page.context().cookies()).find((cookie) => cookie.name === "kw_session");
  expect(session?.httpOnly).toBe(true);
  expect(session?.sameSite).toBe("Lax");
  expect(session?.secure).toBe(true);

  await page.goto("/tasks");
  await page.locator("#task-agent").fill("support");
  await page.getByRole("button", { name: "Create task" }).click();
  await expect(page.getByRole("heading", { name: "Issued token" })).toBeVisible();
  expect(page.url()).not.toContain("kwt_");
  await expect(page.locator("pre").first()).toContainText("kwt_");

  await page.goto("/audit");
  await expect(page.getByText(/Chain intact/)).toBeVisible();
});

test("a down proxy renders an error instead of a blank page", async ({ page }) => {
  const down = process.env.PLAYWRIGHT_DOWN_URL ?? "http://127.0.0.1:3002";
  await page.goto(`${down}/login`);
  await page.locator("#password").fill(password);
  await page.getByRole("button", { name: "Sign In to Operator Console" }).click();
  await expect(page.getByRole("heading", { name: "Proxy unreachable" })).toBeVisible();
});
