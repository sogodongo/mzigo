"use client";

import type { BlastRadiusReport, ScoredConsumer } from "@/lib/types";
import { ImpactBadge } from "./ImpactBadge";

interface Props {
  report: BlastRadiusReport;
}

export function BlastRadiusReportView({ report }: Props) {
  const critical = report.consumers.filter((c) => c.impact === "CRITICAL");
  const high = report.consumers.filter((c) => c.impact === "HIGH");
  const others = report.consumers.filter(
    (c) => c.impact !== "CRITICAL" && c.impact !== "HIGH"
  );

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: "24px" }}>
      {/* Summary banner */}
      <div
        style={{
          background:
            report.worst_impact === "CRITICAL"
              ? "#f43f5e10"
              : report.worst_impact === "HIGH"
              ? "#f59e0b10"
              : "#10b98110",
          border: `1px solid ${
            report.worst_impact === "CRITICAL"
              ? "#f43f5e30"
              : report.worst_impact === "HIGH"
              ? "#f59e0b30"
              : "#10b98130"
          }`,
          borderRadius: "8px",
          padding: "16px 20px",
        }}
      >
        <div style={{ display: "flex", alignItems: "center", gap: "12px", marginBottom: "8px" }}>
          <ImpactBadge level={report.worst_impact} />
          <span style={{ fontSize: "13px", color: "var(--text-secondary)" }}>
            {report.total_consumers_affected} consumer(s) affected
          </span>
        </div>
        <p style={{ margin: 0, fontSize: "13px", color: "var(--text-secondary)", lineHeight: "1.6" }}>
          {report.summary}
        </p>
      </div>

      {/* Required approvals */}
      {report.required_approvals.length > 0 && (
        <div
          style={{
            background: "#f59e0b08",
            border: "1px solid #f59e0b25",
            borderRadius: "8px",
            padding: "14px 18px",
          }}
        >
          <div
            style={{
              fontSize: "11px",
              fontFamily: "'JetBrains Mono', monospace",
              color: "#fbbf24",
              fontWeight: 600,
              letterSpacing: "0.08em",
              marginBottom: "8px",
            }}
          >
            REQUIRED APPROVALS
          </div>
          <div style={{ display: "flex", flexWrap: "wrap", gap: "8px" }}>
            {report.required_approvals.map((team) => (
              <span
                key={team}
                style={{
                  background: "#f59e0b15",
                  color: "#fbbf24",
                  padding: "3px 10px",
                  borderRadius: "4px",
                  fontSize: "12px",
                  fontFamily: "'JetBrains Mono', monospace",
                }}
              >
                {team}
              </span>
            ))}
          </div>
        </div>
      )}

      {/* Consumer table */}
      {report.consumers.length > 0 && (
        <div>
          <div
            style={{
              fontSize: "11px",
              fontFamily: "'JetBrains Mono', monospace",
              color: "var(--text-muted)",
              letterSpacing: "0.08em",
              marginBottom: "12px",
            }}
          >
            AFFECTED CONSUMERS
          </div>
          <div
            style={{
              border: "1px solid var(--border)",
              borderRadius: "8px",
              overflow: "hidden",
            }}
          >
            <table style={{ width: "100%", borderCollapse: "collapse" }}>
              <thead>
                <tr style={{ background: "var(--bg-surface)" }}>
                  {["Consumer", "Type", "Team", "Impact", "Affected Fields", "Depth"].map((h) => (
                    <th
                      key={h}
                      style={{
                        padding: "10px 16px",
                        textAlign: "left",
                        fontSize: "11px",
                        fontFamily: "'JetBrains Mono', monospace",
                        color: "var(--text-muted)",
                        letterSpacing: "0.06em",
                        fontWeight: 500,
                        borderBottom: "1px solid var(--border)",
                      }}
                    >
                      {h}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {report.consumers.map((consumer, i) => (
                  <ConsumerRow key={consumer.name} consumer={consumer} index={i} />
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}

function ConsumerRow({ consumer, index }: { consumer: ScoredConsumer; index: number }) {
  return (
    <tr
      style={{
        borderBottom: "1px solid var(--border)",
        background: index % 2 === 0 ? "transparent" : "var(--bg-surface)",
      }}
    >
      <td style={{ padding: "10px 16px" }}>
        <span
          style={{
            fontFamily: "'JetBrains Mono', monospace",
            fontSize: "12px",
            color: "var(--text-primary)",
          }}
        >
          {consumer.name}
        </span>
        {consumer.is_direct && (
          <span
            style={{
              marginLeft: "8px",
              fontSize: "10px",
              color: "var(--accent)",
              fontFamily: "'JetBrains Mono', monospace",
            }}
          >
            direct
          </span>
        )}
      </td>
      <td style={{ padding: "10px 16px" }}>
        <span style={{ fontSize: "11px", color: "var(--text-muted)", fontFamily: "monospace" }}>
          {consumer.node_type}
        </span>
      </td>
      <td style={{ padding: "10px 16px" }}>
        <span style={{ fontSize: "12px", color: "var(--text-secondary)" }}>
          {consumer.owner_team ?? "unknown"}
        </span>
      </td>
      <td style={{ padding: "10px 16px" }}>
        <ImpactBadge level={consumer.impact} />
      </td>
      <td style={{ padding: "10px 16px" }}>
        {consumer.affected_fields.length > 0 ? (
          <div style={{ display: "flex", flexWrap: "wrap", gap: "4px" }}>
            {consumer.affected_fields.map((f) => (
              <span
                key={f}
                style={{
                  background: "#f43f5e15",
                  color: "#fb7185",
                  padding: "1px 6px",
                  borderRadius: "3px",
                  fontSize: "11px",
                  fontFamily: "'JetBrains Mono', monospace",
                }}
              >
                {f}
              </span>
            ))}
          </div>
        ) : (
          <span style={{ fontSize: "11px", color: "var(--text-muted)" }}>none</span>
        )}
      </td>
      <td style={{ padding: "10px 16px" }}>
        <span style={{ fontSize: "12px", color: "var(--text-muted)", fontFamily: "monospace" }}>
          {consumer.depth}
        </span>
      </td>
    </tr>
  );
}
