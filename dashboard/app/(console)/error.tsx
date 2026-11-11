"use client";

export default function ConsoleError({ reset }: { error: Error; reset: () => void }) {
  return (
    <>
      <h1>Console unavailable</h1>
      <p className="error">This page could not be rendered.</p>
      <div className="row">
        <button type="button" onClick={() => reset()}>
          Try again
        </button>
      </div>
    </>
  );
}
