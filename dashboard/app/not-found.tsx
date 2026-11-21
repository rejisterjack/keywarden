import Link from "next/link";

export default function NotFound() {
  return (
    <main className="login-shell">
      <h1>Page not found</h1>
      <p className="lede">This URL is not part of the Keywarden dashboard.</p>
      <div className="row">
        <Link href="/">Home</Link>
        <Link href="/login">Operator console</Link>
      </div>
    </main>
  );
}