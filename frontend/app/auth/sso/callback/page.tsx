"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { tokens } from "@/lib/api";
import { Spinner } from "@/components/ui/misc";

export default function SsoCallbackPage() {
  const router = useRouter();
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    // Backend redirects here with tokens in the URL fragment (never sent to a server).
    const frag = new URLSearchParams(window.location.hash.replace(/^#/, ""));
    const access = frag.get("access_token");
    const refresh = frag.get("refresh_token");
    const err = frag.get("error");

    if (err) {
      setError(err);
      return;
    }
    if (access && refresh) {
      tokens.set(access, refresh);
      window.location.hash = "";
      router.replace("/dashboard");
    } else {
      setError("Missing tokens in SSO response");
    }
  }, [router]);

  return (
    <div className="flex min-h-screen flex-col items-center justify-center gap-3">
      {error ? (
        <>
          <p className="text-danger">Sign-in failed: {error}</p>
          <a href="/login" className="text-sm text-primary hover:underline">Back to login</a>
        </>
      ) : (
        <>
          <Spinner className="h-6 w-6" />
          <p className="text-sm text-muted">Completing sign-in…</p>
        </>
      )}
    </div>
  );
}
