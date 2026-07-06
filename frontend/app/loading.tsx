"use client";

import { motion } from "motion/react";
import { BRAND } from "@/lib/brand";

export default function Loading() {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center gap-5">
      <motion.div
        animate={{ scale: [1, 1.08, 1], opacity: [0.85, 1, 0.85] }}
        transition={{ duration: 1.4, repeat: Infinity, ease: "easeInOut" }}
      >
        {/* eslint-disable-next-line @next/next/no-img-element */}
        <img src={BRAND.logo.full} alt={BRAND.company.shortName} width={44} height={44} />
      </motion.div>

      <div className="h-1 w-40 overflow-hidden rounded-full bg-surface-2">
        <motion.div
          className="h-full w-1/3 rounded-full bg-primary"
          animate={{ x: ["-140%", "420%"] }}
          transition={{ duration: 1.1, repeat: Infinity, ease: "easeInOut" }}
        />
      </div>

      <p className="font-mono text-[11px] uppercase tracking-[0.2em] text-muted">Loading</p>
    </div>
  );
}
