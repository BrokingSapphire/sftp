"use client";

import { AnimatePresence, motion } from "motion/react";
import { X } from "lucide-react";
import { Logo } from "@/components/logo";
import { SidebarNav } from "@/components/app-shell/sidebar";

/** Slide-in navigation drawer for small screens. */
export function MobileNav({ open, onClose }: { open: boolean; onClose: () => void }) {
  return (
    <AnimatePresence>
      {open && (
        <div className="fixed inset-0 z-40 md:hidden">
          <motion.div
            initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}
            onClick={onClose}
            className="absolute inset-0 bg-black/50 backdrop-blur-sm"
          />
          <motion.aside
            initial={{ x: "-100%" }} animate={{ x: 0 }} exit={{ x: "-100%" }}
            transition={{ type: "spring", stiffness: 320, damping: 34 }}
            className="absolute left-0 top-0 flex h-full w-72 max-w-[85%] flex-col border-r border-border bg-surface"
          >
            <div className="flex h-16 items-center justify-between px-5">
              <Logo size={28} />
              <button onClick={onClose} aria-label="Close menu" className="text-muted hover:text-foreground">
                <X size={20} />
              </button>
            </div>
            <SidebarNav onNavigate={onClose} />
          </motion.aside>
        </div>
      )}
    </AnimatePresence>
  );
}
