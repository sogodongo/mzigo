import Link from "next/link";
import type { Contract } from "@/lib/types";
import { StatusBadge } from "./ImpactBadge";

interface Props {
  contract: Contract;
  activeVersion?: { version: string; status: string };
}

export function ContractCard({ contract, activeVersion }: Props) {
  return (
    <Link
      href={`/contracts/${encodeURIComponent(contract.name)}`}
      style={{ textDecoration: "none", display: "block" }}
    >
      <div
        style={{
          background: "var(--bg-surface)",
          border: "1px solid var(--border)",
          borderRadius: "8px",
          padding: "16px 20px",
          cursor: "pointer",
          transition: "border-color 0.15s, background 0.15s",
        }}
        className="contract-card"
      >
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start", marginBottom: "8px" }}>
          <span
            style={{
              fontFamily: "'JetBrains Mono', monospace",
              fontSize: "13px",
              fontWeight: 600,
              color: "var(--accent)",
            }}
          >
            {contract.name}
          </span>
          {activeVersion && <StatusBadge status={activeVersion.status} />}
        </div>

        <div
          style={{
            fontSize: "11px",
            color: "var(--text-muted)",
            fontFamily: "'JetBrains Mono', monospace",
            marginBottom: "10px",
          }}
        >
          {contract.topic}
        </div>

        {contract.description && (
          <p
            style={{
              fontSize: "13px",
              color: "var(--text-secondary)",
              margin: "0 0 12px",
              lineHeight: "1.5",
            }}
          >
            {contract.description}
          </p>
        )}

        <div style={{ display: "flex", alignItems: "center", gap: "16px" }}>
          <span
            style={{
              fontSize: "11px",
              color: "var(--text-muted)",
              background: "var(--bg-raised)",
              padding: "2px 8px",
              borderRadius: "4px",
              fontFamily: "'JetBrains Mono', monospace",
            }}
          >
            {contract.owner_team}
          </span>
          {activeVersion && (
            <span
              style={{
                fontSize: "11px",
                color: "var(--text-muted)",
                fontFamily: "'JetBrains Mono', monospace",
              }}
            >
              v{activeVersion.version}
            </span>
          )}
        </div>
      </div>

      <style>{`
        .contract-card:hover {
          border-color: var(--border-dim);
          background: var(--bg-raised);
        }
      `}</style>
    </Link>
  );
}
