import { describe, it, expect } from "vitest";
import { cn, formatBytes, timeAgo } from "./utils";

describe("cn", () => {
  it("merges and dedupes tailwind classes", () => {
    expect(cn("p-2", "p-4")).toBe("p-4");
    expect(cn("text-sm", false && "hidden", "font-medium")).toBe("text-sm font-medium");
  });
});

describe("formatBytes", () => {
  it("formats sizes", () => {
    expect(formatBytes(0)).toBe("0 B");
    expect(formatBytes(1024)).toBe("1 KB");
    expect(formatBytes(1536)).toBe("1.5 KB");
    expect(formatBytes(1024 * 1024)).toBe("1 MB");
    expect(formatBytes(5 * 1024 * 1024 * 1024)).toBe("5 GB");
  });
  it("handles negatives/zero", () => {
    expect(formatBytes(-5)).toBe("0 B");
  });
});

describe("timeAgo", () => {
  it("returns empty for undefined", () => {
    expect(timeAgo(undefined)).toBe("");
  });
  it("returns relative labels", () => {
    const now = new Date();
    expect(timeAgo(now.toISOString())).toBe("just now");
    expect(timeAgo(new Date(now.getTime() - 5 * 60000).toISOString())).toBe("5m ago");
    expect(timeAgo(new Date(now.getTime() - 3 * 3600000).toISOString())).toBe("3h ago");
  });
});
