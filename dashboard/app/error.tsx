"use client";

import Link from "next/link";
import { useEffect } from "react";

export default function RootError({
  retry,
}: {
  error: unknown;
  retry: () => void;
}) {
  useEffect(() => {
    document.title = "Error — Keywarden";
  }, []);

  return (
    <main className="login-shell">
      <h1>Something went wrong</h1>
      <p className="lede">This page could not be rendered. The details stay on the server.</p>
      <div className="row">
        <button type="button" onClick={() => retry()}>
          Try again
        </button>
        <Link href="/">Home</Link>
      </div>
    </main>
  );
}
