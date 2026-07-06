"use client";

import { useEffect, useLayoutEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { AnimatePresence, motion } from "motion/react";

export interface MenuItem {
  label: string;
  icon?: React.ElementType;
  onClick?: () => void;
  danger?: boolean;
  separator?: boolean;
  disabled?: boolean;
}

export interface MenuState {
  x: number;
  y: number;
  items: MenuItem[];
}

/** Hook that manages a single right-click menu. */
export function useContextMenu() {
  const [menu, setMenu] = useState<MenuState | null>(null);
  const open = (e: React.MouseEvent, items: MenuItem[]) => {
    e.preventDefault();
    e.stopPropagation();
    setMenu({ x: e.clientX, y: e.clientY, items });
  };
  return { menu, open, close: () => setMenu(null) };
}

/** Floating context menu rendered in a portal, clamped to the viewport. */
export function ContextMenu({ menu, onClose }: { menu: MenuState | null; onClose: () => void }) {
  const ref = useRef<HTMLDivElement>(null);
  const [pos, setPos] = useState({ x: 0, y: 0 });
  const [mounted, setMounted] = useState(false);
  useEffect(() => setMounted(true), []);

  useLayoutEffect(() => {
    if (!menu || !ref.current) return;
    const { offsetWidth: w, offsetHeight: h } = ref.current;
    const x = Math.min(menu.x, window.innerWidth - w - 8);
    const y = Math.min(menu.y, window.innerHeight - h - 8);
    setPos({ x, y });
  }, [menu]);

  useEffect(() => {
    if (!menu) return;
    const close = () => onClose();
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && onClose();
    window.addEventListener("click", close);
    window.addEventListener("scroll", close, true);
    window.addEventListener("resize", close);
    window.addEventListener("keydown", onKey);
    window.addEventListener("contextmenu", close);
    return () => {
      window.removeEventListener("click", close);
      window.removeEventListener("scroll", close, true);
      window.removeEventListener("resize", close);
      window.removeEventListener("keydown", onKey);
      window.removeEventListener("contextmenu", close);
    };
  }, [menu, onClose]);

  if (!mounted) return null;

  return createPortal(
    <AnimatePresence>
      {menu && (
        <motion.div
          ref={ref}
          initial={{ opacity: 0, scale: 0.96, y: -4 }}
          animate={{ opacity: 1, scale: 1, y: 0 }}
          exit={{ opacity: 0, scale: 0.96 }}
          transition={{ duration: 0.12, ease: [0.22, 1, 0.36, 1] }}
          style={{ left: pos.x, top: pos.y }}
          onContextMenu={(e) => e.preventDefault()}
          className="fixed z-[60] min-w-52 overflow-hidden rounded-xl border border-border bg-surface p-1 shadow-xl"
        >
          {menu.items.map((it, i) =>
            it.separator ? (
              <div key={i} className="my-1 h-px bg-border" />
            ) : (
              <button
                key={i}
                disabled={it.disabled}
                onClick={() => { onClose(); it.onClick?.(); }}
                className={`flex w-full items-center gap-2.5 rounded-lg px-2.5 py-1.5 text-left text-sm transition-colors disabled:opacity-40 ${
                  it.danger ? "text-danger hover:bg-danger/10" : "hover:bg-surface-2"
                }`}
              >
                {it.icon && <it.icon size={15} className="shrink-0" />}
                {it.label}
              </button>
            ),
          )}
        </motion.div>
      )}
    </AnimatePresence>,
    document.body,
  );
}
