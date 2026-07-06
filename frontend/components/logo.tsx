/* eslint-disable @next/next/no-img-element */
import { cn } from "@/lib/utils";

interface LogoProps {
  size?: number;
  withWordmark?: boolean;
  className?: string;
}

/** Sapphire brand mark (real gradient gem) with optional wordmark. */
export function Logo({ size = 28, withWordmark = true, className }: LogoProps) {
  return (
    <div className={cn("flex items-center gap-2.5", className)}>
      <img src="/logo.svg" alt="Sapphire" width={size} height={size} className="shrink-0 select-none" draggable={false} />
      {withWordmark && (
        <div className="flex flex-col leading-none">
          <span className="text-[15px] font-semibold tracking-tight">Sapphire</span>
          <span className="mt-0.5 font-mono text-[10px] uppercase tracking-[0.2em] text-muted">SFTP</span>
        </div>
      )}
    </div>
  );
}
