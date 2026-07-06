"use client";

import { motion, type Variants, type HTMLMotionProps } from "motion/react";
import { cn } from "@/lib/utils";

/** Springy, tactile easing used across the app. */
export const spring = { type: "spring", stiffness: 380, damping: 30 } as const;
export const easeOut = [0.22, 1, 0.36, 1] as const;

/** Container that staggers its children into view. */
export const stagger: Variants = {
  hidden: {},
  show: { transition: { staggerChildren: 0.05, delayChildren: 0.04 } },
};

/** A single item that rises + fades in. */
export const rise: Variants = {
  hidden: { opacity: 0, y: 10 },
  show: { opacity: 1, y: 0, transition: { duration: 0.4, ease: easeOut } },
};

/** Fade + rise wrapper. */
export function FadeIn({ className, children, delay = 0, ...props }: HTMLMotionProps<"div"> & { delay?: number }) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.4, ease: easeOut, delay }}
      className={className}
      {...props}
    >
      {children}
    </motion.div>
  );
}

/** Staggered list container. */
export function StaggerList({ className, children, ...props }: HTMLMotionProps<"div">) {
  return (
    <motion.div variants={stagger} initial="hidden" animate="show" className={className} {...props}>
      {children}
    </motion.div>
  );
}

/** Staggered list row. */
export function StaggerItem({ className, children, ...props }: HTMLMotionProps<"div">) {
  return (
    <motion.div variants={rise} className={className} {...props}>
      {children}
    </motion.div>
  );
}

/** A pressable surface with hover-lift + tap-shrink micro-interaction. */
export function Pressable({ className, children, ...props }: HTMLMotionProps<"div">) {
  return (
    <motion.div
      whileHover={{ y: -2 }}
      whileTap={{ scale: 0.98 }}
      transition={spring}
      className={cn(className)}
      {...props}
    >
      {children}
    </motion.div>
  );
}

export { motion };
