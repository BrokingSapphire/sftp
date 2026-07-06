"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { motion } from "motion/react";
import { authApi } from "@/lib/endpoints";

const FACTS = [
  "All your files live on-premise — nothing ever touches the cloud.",
  "Every action is recorded in an immutable, compliance-grade audit trail.",
  "Interrupted uploads resume automatically, right where they left off.",
  "Files are addressed by opaque keys — path traversal is impossible.",
  "Passwords are protected with Argon2id, the modern hashing standard.",
  "Share links can expire, require a password, and cap total downloads.",
  "Native SFTP and REST share the same accounts and the same storage.",
  "Deleted files rest in the recycle bin before they are purged.",
  "Range requests mean instant seeking in large video previews.",
  "Role-based access control governs every single file operation.",
];

export default function LogoutPage() {
  const router = useRouter();
  const [progress, setProgress] = useState(0);
  const [fact, setFact] = useState(0);

  useEffect(() => {
    authApi.logout().catch(() => {});

    const started = performance.now();
    const duration = 2600;
    let raf = 0;
    const tick = () => {
      const p = Math.min(100, ((performance.now() - started) / duration) * 100);
      setProgress(p);
      if (p < 100) raf = requestAnimationFrame(tick);
      else setTimeout(() => router.replace("/login"), 250);
    };
    raf = requestAnimationFrame(tick);

    const factTimer = setInterval(() => setFact((f) => (f + 1) % FACTS.length), 1300);
    return () => { cancelAnimationFrame(raf); clearInterval(factTimer); };
  }, [router]);

  return (
    <div className="relative flex min-h-screen flex-col items-center justify-center overflow-hidden bg-[#053e42] px-6 text-white">
      <div
        aria-hidden
        className="pointer-events-none absolute inset-0 opacity-[0.06]"
        style={{
          backgroundImage:
            "linear-gradient(to right, #fff 1px, transparent 1px), linear-gradient(to bottom, #fff 1px, transparent 1px)",
          backgroundSize: "44px 44px",
        }}
      />
      <motion.div
        aria-hidden
        className="pointer-events-none absolute -top-24 left-1/2 h-96 w-[42rem] -translate-x-1/2 rounded-full bg-teal-400/10 blur-3xl"
        animate={{ scale: [1, 1.15, 1], opacity: [0.4, 0.7, 0.4] }}
        transition={{ duration: 6, repeat: Infinity, ease: "easeInOut" }}
      />

      <motion.div initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.4 }}>
        <div className="flex items-center gap-2.5">
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img src="/logo.svg" alt="Sapphire" width={40} height={40} className="drop-shadow" />
          <span className="text-xl font-semibold tracking-tight">Sapphire</span>
        </div>
      </motion.div>

      <p className="mt-8 font-mono text-xs uppercase tracking-[0.24em] text-teal-200/70">Signing you out</p>

      {/* Progress bar */}
      <div className="mt-4 h-1 w-72 overflow-hidden rounded-full bg-white/15">
        <div className="h-full rounded-full bg-teal-300 transition-[width] duration-75" style={{ width: `${progress}%` }} />
      </div>
      <span className="mt-2 font-mono text-[11px] text-teal-200/50">{Math.round(progress)}%</span>

      {/* Rotating facts */}
      <div className="mt-10 h-12 max-w-md text-center">
        <motion.p
          key={fact}
          initial={{ opacity: 0, y: 8 }}
          animate={{ opacity: 1, y: 0 }}
          exit={{ opacity: 0 }}
          transition={{ duration: 0.4 }}
          className="text-sm leading-relaxed text-teal-50/85"
        >
          <span className="mr-2 font-mono text-[10px] uppercase tracking-wider text-teal-300/70">Did you know</span>
          <br />
          {FACTS[fact]}
        </motion.p>
      </div>

      <p className="absolute bottom-8 font-mono text-[11px] text-teal-200/40">Sapphire SFTP · on-premise file transfer</p>
    </div>
  );
}
