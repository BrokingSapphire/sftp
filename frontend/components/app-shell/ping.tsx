"use client";

import { useEffect, useRef, useState } from "react";
import { motion } from "motion/react";
import { cn } from "@/lib/utils";

interface PingState {
  ms: number | null;
  up: boolean;
}

/** Continuously pings the server health endpoint and shows round-trip latency. */
export function PingIndicator() {
  const [state, setState] = useState<PingState>({ ms: null, up: true });
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    let alive = true;

    async function ping() {
      const start = performance.now();
      try {
        const res = await fetch("/api/v1/health-check", { cache: "no-store" });
        const ms = Math.round(performance.now() - start);
        if (alive) setState({ ms, up: res.ok });
      } catch {
        if (alive) setState({ ms: null, up: false });
      }
      if (alive) timer.current = setTimeout(ping, 3000);
    }
    ping();

    return () => {
      alive = false;
      if (timer.current) clearTimeout(timer.current);
    };
  }, []);

  const { ms, up } = state;
  const tone = !up ? "down" : ms == null ? "wait" : ms < 120 ? "good" : ms < 350 ? "ok" : "slow";
  const color = {
    good: "text-success",
    ok: "text-warning",
    slow: "text-danger",
    down: "text-danger",
    wait: "text-muted",
  }[tone];
  const dot = {
    good: "bg-success",
    ok: "bg-warning",
    slow: "bg-danger",
    down: "bg-danger",
    wait: "bg-muted",
  }[tone];

  return (
    <div
      title={up ? `Server latency: ${ms ?? "…"} ms` : "Server unreachable"}
      className="flex items-center gap-1.5 rounded-md border border-border px-2 py-1.5 font-mono text-xs tabular-nums"
    >
      <span className="relative flex h-2 w-2">
        {up && (
          <motion.span
            className={cn("absolute inline-flex h-full w-full rounded-full opacity-60", dot)}
            animate={{ scale: [1, 2.2, 1], opacity: [0.6, 0, 0.6] }}
            transition={{ duration: 2, repeat: Infinity, ease: "easeInOut" }}
          />
        )}
        <span className={cn("relative inline-flex h-2 w-2 rounded-full", dot)} />
      </span>
      <span className={cn("hidden sm:inline", color)}>
        {up ? (ms == null ? "…" : `${ms} ms`) : "offline"}
      </span>
    </div>
  );
}
