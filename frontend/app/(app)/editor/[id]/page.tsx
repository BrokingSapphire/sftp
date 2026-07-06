"use client";

import { useEffect, useRef, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { ArrowLeft, Loader2, FileWarning } from "lucide-react";
import { editorApi } from "@/lib/endpoints";
import { Button } from "@/components/ui/button";

/* eslint-disable @typescript-eslint/no-explicit-any */
declare global {
  interface Window { DocsAPI?: any }
}

// Loads a script once and resolves when ready.
function loadScript(src: string): Promise<void> {
  return new Promise((resolve, reject) => {
    if (document.querySelector(`script[src="${src}"]`)) return resolve();
    const s = document.createElement("script");
    s.src = src; s.async = true;
    s.onload = () => resolve();
    s.onerror = () => reject(new Error("failed to load editor"));
    document.body.appendChild(s);
  });
}

export default function OfficeEditorPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const mounted = useRef(false);

  useEffect(() => {
    if (mounted.current) return;
    mounted.current = true;
    (async () => {
      try {
        const { doc_server_url, config } = await editorApi.config(id);
        await loadScript(`${doc_server_url.replace(/\/$/, "")}/web-apps/apps/api/documents/api.js`);
        if (!window.DocsAPI) throw new Error("editor unavailable");
        // eslint-disable-next-line no-new
        new window.DocsAPI.DocEditor("onlyoffice-editor", {
          ...config,
          width: "100%",
          height: "100%",
          events: { onError: () => setError("The editor reported an error.") },
        });
        setLoading(false);
      } catch (e) {
        setError((e as { message?: string })?.message || "Could not open the editor");
        setLoading(false);
      }
    })();
  }, [id]);

  return (
    <div className="fixed inset-0 z-50 flex flex-col bg-background">
      <div className="flex h-12 items-center gap-2 border-b border-border px-4">
        <Button variant="ghost" size="sm" onClick={() => router.back()}><ArrowLeft size={16} /> Back</Button>
        <span className="text-sm font-medium text-muted">Office editor</span>
      </div>

      <div className="relative flex-1">
        {loading && (
          <div className="absolute inset-0 flex items-center justify-center gap-2 text-muted">
            <Loader2 className="animate-spin" size={20} /> Opening…
          </div>
        )}
        {error && (
          <div className="absolute inset-0 flex flex-col items-center justify-center gap-3 text-center text-muted">
            <FileWarning size={40} />
            <p className="max-w-sm text-sm">{error}</p>
            <Button size="sm" variant="outline" onClick={() => router.back()}>Go back</Button>
          </div>
        )}
        <div id="onlyoffice-editor" className="h-full w-full" />
      </div>
    </div>
  );
}
