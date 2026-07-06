"use client";

import { motion } from "motion/react";

/**
 * A friendly, animated empty state — floating illustration, a title, and a bit
 * of humour. Reused across the app so empty sections feel intentional, not broken.
 */
export function EmptyState({
  icon: Icon,
  title,
  subtitle,
  action,
  className,
}: {
  icon: React.ElementType;
  title: string;
  subtitle?: string;
  action?: React.ReactNode;
  className?: string;
}) {
  return (
    <div className={`flex min-h-[18rem] flex-col items-center justify-center gap-4 rounded-2xl border border-dashed border-border bg-surface px-6 py-12 text-center ${className ?? ""}`}>
      {/* Floating illustration: layered glow + gently bobbing icon */}
      <div className="relative">
        <motion.div
          aria-hidden
          className="absolute inset-0 rounded-full bg-primary/20 blur-2xl"
          animate={{ scale: [1, 1.15, 1], opacity: [0.4, 0.7, 0.4] }}
          transition={{ duration: 4, repeat: Infinity, ease: "easeInOut" }}
        />
        <motion.div
          className="relative flex h-20 w-20 items-center justify-center rounded-2xl bg-gradient-to-br from-primary/15 to-primary/5 text-primary shadow-sm"
          animate={{ y: [0, -8, 0], rotate: [-2, 2, -2] }}
          transition={{ duration: 5, repeat: Infinity, ease: "easeInOut" }}
        >
          <Icon size={36} strokeWidth={1.6} />
        </motion.div>
      </div>

      <div className="space-y-1">
        <p className="text-base font-semibold">{title}</p>
        {subtitle && <p className="mx-auto max-w-sm text-sm text-muted">{subtitle}</p>}
      </div>

      {action}
    </div>
  );
}
