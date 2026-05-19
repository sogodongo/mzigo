"use client";

import { useState } from "react";
import { BlastRadiusReportView } from "@/components/BlastRadiusReport";
import { ClassificationBadge } from "@/components/ImpactBadge";
import type { BlastRadiusReport } from "@/lib/types";

export default function BlastRadiusPage() {
  const [topic, setTopic] = useState("");
  const [fieldsInput, setFieldsInput] = useState("");
  const [report, setReport] = useState<BlastRadiusReport | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleAnalyze() {
    const fields = fieldsInput
      .split(",")
      .map((f) => f.trim())
      .filter(Boolean);

    if (!topic || fields.length === 0) {
      setError("Topic and at least one changed field are required.");
      return;
    }

    setLoading(true);
    setError(null);
    setReport(null);

    try {
      const res = await fetch("/api/blast-radius", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ topic, changed_fields: fields }),
      });

      if (!res.ok) throw new Error(`Analysis failed: ${res.status}`);
      const data = await res.json();
      setReport(data);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Analysis failed.");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div style={{ padding: "32px 40px", maxWidth: "900px" }}>
      <div style={{ marginBottom: "32px" }}>
        <h1
          style={{
            fontSize: "22px",
            fontWeight: 600,
            color: "var(--text-primary)",
            margin: "0 0 6px",
          }}
        >
          Blast Radius
        </h1>
        <p style={{ margin: 0, fontSize: "13px", color: "var(--text-muted)" }}>
          Compute the downstream impact of a proposed schema change before it merges
        </p>
      </div>

      {/* Input form */}
      <div
        style={{
          background: "var(--bg-surface)",
          border: "1px solid var(--border)",
          borderRadius: "10px",
          padding: "24px",
          marginBottom: "28px",
        }}
      >
        <div style={{ display: "flex", flexDirection: "column", gap: "16px" }}>
          <label style={{ display: "flex", flexDirection: "column", gap: "6px" }}>
            <span
              style={{
                fontSize: "11px",
                fontFamily: "'JetBrains Mono', monospace",
                color: "var(--text-muted)",
                letterSpacing: "0.08em",
              }}
            >
              TOPIC
            </span>
            <input
              type="text"
              value={topic}
              onChange={(e) => setTopic(e.target.value)}
              placeholder="payments.transactions"
              style={{
                background: "var(--bg-raised)",
                border: "1px solid var(--border-dim)",
                borderRadius: "6px",
                padding: "9px 12px",
                color: "var(--text-primary)",
                fontFamily: "'JetBrains Mono', monospace",
                fontSize: "13px",
                outline: "none",
              }}
            />
          </label>

          <label style={{ display: "flex", flexDirection: "column", gap: "6px" }}>
            <span
              style={{
                fontSize: "11px",
                fontFamily: "'JetBrains Mono', monospace",
                color: "var(--text-muted)",
                letterSpacing: "0.08em",
              }}
            >
              CHANGED FIELDS (comma-separated)
            </span>
            <input
              type="text"
              value={fieldsInput}
              onChange={(e) => setFieldsInput(e.target.value)}
              placeholder="amount_cents, currency"
              style={{
                background: "var(--bg-raised)",
                border: "1px solid var(--border-dim)",
                borderRadius: "6px",
                padding: "9px 12px",
                color: "var(--text-primary)",
                fontFamily: "'JetBrains Mono', monospace",
                fontSize: "13px",
                outline: "none",
              }}
            />
          </label>

          <div style={{ display: "flex", alignItems: "center", gap: "12px" }}>
            <button
              onClick={handleAnalyze}
              disabled={loading}
              style={{
                background: loading ? "var(--bg-raised)" : "var(--accent)",
                color: loading ? "var(--text-muted)" : "#000",
                border: "none",
                borderRadius: "6px",
                padding: "9px 20px",
                fontSize: "13px",
                fontWeight: 600,
                cursor: loading ? "not-allowed" : "pointer",
                fontFamily: "'DM Sans', sans-serif",
                transition: "opacity 0.15s",
              }}
            >
              {loading ? "Analyzing..." : "Analyze Impact"}
            </button>

            {error && (
              <span
                style={{
                  fontSize: "12px",
                  color: "#fb7185",
                  fontFamily: "'JetBrains Mono', monospace",
                }}
              >
                {error}
              </span>
            )}
          </div>
        </div>
      </div>

      {/* Report */}
      {report && (
        <div style={{ animation: "fadeIn 0.3s ease-out" }}>
          <div
            style={{
              display: "flex",
              alignItems: "center",
              gap: "12px",
              marginBottom: "20px",
            }}
          >
            <span style={{ fontSize: "13px", color: "var(--text-muted)" }}>
              Analysis for
            </span>
            <span
              style={{
                fontFamily: "'JetBrains Mono', monospace",
                fontSize: "13px",
                color: "var(--accent)",
              }}
            >
              {report.topic}
            </span>
          </div>
          <BlastRadiusReportView report={report} />
        </div>
      )}
    </div>
  );
}
