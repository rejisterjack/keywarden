export function ProxyDown({ message }: { message: string }) {
  return (
    <>
      <h1>Proxy unreachable</h1>
      <p className="error">{message}</p>
      <p className="lede">
        The console calls the proxy with the admin token. Check KEYWARDEN_API_URL and that the proxy is listening.
      </p>
    </>
  );
}
