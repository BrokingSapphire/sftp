"use client";

/* eslint-disable @next/next/no-img-element */
import { useState } from "react";
import { avatarApi } from "@/lib/endpoints";
import { cn } from "@/lib/utils";

function initials(name: string) {
  return (name || "?").trim().split(/\s+/).map((s) => s[0]).slice(0, 2).join("").toUpperCase();
}

/** User avatar: shows the profile photo when available, else colourful initials. */
export function Avatar({
  userId, name, hasAvatar, size = 32, className,
}: {
  userId?: string; name: string; hasAvatar?: boolean; size?: number; className?: string;
}) {
  const [errored, setErrored] = useState(false);
  const showImage = hasAvatar && userId && !errored;

  return (
    <span
      className={cn("relative inline-flex shrink-0 items-center justify-center overflow-hidden rounded-full bg-primary/15 font-semibold text-primary", className)}
      style={{ width: size, height: size, fontSize: Math.round(size * 0.38) }}
    >
      {showImage ? (
        <img
          src={avatarApi.url(userId!)}
          alt={name}
          width={size}
          height={size}
          className="h-full w-full object-cover"
          onError={() => setErrored(true)}
          draggable={false}
        />
      ) : (
        initials(name)
      )}
    </span>
  );
}
