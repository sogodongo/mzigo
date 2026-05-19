import type { ImpactLevel, ChangeClassification } from "@/lib/types";
import clsx from "clsx";

const IMPACT_STYLES: Record<ImpactLevel, { bg: string; color: string; label: string }> = {
  CRITICAL: { bg: "#f43f5e18", color: "#fb7185", label: "CRITICAL" },
  HIGH:     { bg: "#f59e0b18", color: "#fbbf24", label: "HIGH" },
  MEDIUM:   { bg: "#3b82f618", color: "#60a5fa", label: "MEDIUM" },
  NONE:     { bg: "#33415520", color: "#64748b", label: "NONE" },
};

const CLASS_STYLES: Record<ChangeClassification, { bg: string; color: string }> = {
  BREAKING:   { bg: "#f43f5e18", color: "#fb7185" },
  COMPATIBLE: { bg: "#f59e0b18", color: "#fbbf24" },
  SAFE:       { bg: "#10b98118", color: "#34d399" },
};

export function ImpactBadge({ level }: { level: ImpactLevel }) {
  const s = IMPACT_STYLES[level];
  return (
    <span
      style={{
        background: s.bg,
        color: s.color,
        padding: "2px 8px",
        borderRadius: "4px",
        fontSize: "11px",
        fontFamily: "'JetBrains Mono', monospace",
        fontWeight: 600,
        letterSpacing: "0.06em",
        display: "inline-block",
      }}
    >
      {s.label}
    </span>
  );
}

export function ClassificationBadge({ value }: { value: ChangeClassification }) {
  const s = CLASS_STYLES[value];
  return (
    <span
      style={{
        background: s.bg,
        color: s.color,
        padding: "2px 10px",
        borderRadius: "4px",
        fontSize: "11px",
        fontFamily: "'JetBrains Mono', monospace",
        fontWeight: 600,
        letterSpacing: "0.06em",
        display: "inline-block",
      }}
    >
      {value}
    </span>
  );
}

export function StatusBadge({ status }: { status: string }) {
  const colors: Record<string, { bg: string; color: string }> = {
    ACTIVE:           { bg: "#10b98118", color: "#34d399" },
    DEPRECATED:       { bg: "#33415530", color: "#64748b" },
    DRAFT:            { bg: "#3b82f618", color: "#60a5fa" },
    PENDING_APPROVAL: { bg: "#f59e0b18", color: "#fbbf24" },
  };
  const s = colors[status] ?? { bg: "#33415530", color: "#94a3b8" };
  return (
    <span
      style={{
        background: s.bg,
        color: s.color,
        padding: "2px 8px",
        borderRadius: "4px",
        fontSize: "11px",
        fontFamily: "'JetBrains Mono', monospace",
        fontWeight: 500,
        letterSpacing: "0.04em",
        display: "inline-block",
      }}
    >
      {status}
    </span>
  );
}
