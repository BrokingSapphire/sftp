/* eslint-disable @next/next/no-img-element */
import { cn } from "@/lib/utils";
import { BRAND } from "@/lib/brand";

interface LogoProps {
  size?: number;
  withWordmark?: boolean;
  className?: string;
}

/** Brand mark + optional wordmark, all driven by brand.config.json. */
export function Logo({ size = 28, withWordmark = true, className }: LogoProps) {
  return (
    <div className={cn("flex items-center gap-2.5", className)}>
      <img
        src={BRAND.logo.full}
        alt={BRAND.company.shortName}
        width={size}
        height={size}
        className="shrink-0 select-none"
        draggable={false}
      />
      {withWordmark && (
        <div className="flex flex-col leading-none">
          <span className="text-[15px] font-semibold tracking-tight">{BRAND.company.shortName}</span>
          <span className="mt-0.5 font-mono text-[10px] uppercase tracking-[0.2em] text-muted">{BRAND.company.productShort}</span>
        </div>
      )}
    </div>
  );
}
