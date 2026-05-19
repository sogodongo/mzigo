import { listContracts } from "@/lib/api";
import { ContractCard } from "@/components/ContractCard";

export const dynamic = "force-dynamic";

export default async function ContractsPage() {
  let contracts = [];
  let error: string | null = null;

  try {
    contracts = await listContracts();
  } catch (e) {
    error = "Could not reach the contracts service.";
  }

  return (
    <div style={{ padding: "32px 40px" }}>
      {/* Page header */}
      <div style={{ marginBottom: "32px" }}>
        <h1
          style={{
            fontSize: "22px",
            fontWeight: 600,
            color: "var(--text-primary)",
            margin: "0 0 6px",
            fontFamily: "'DM Sans', sans-serif",
          }}
        >
          Contracts
        </h1>
        <p style={{ margin: 0, fontSize: "13px", color: "var(--text-muted)" }}>
          Active data contracts enforced at the gateway
        </p>
      </div>

      {error ? (
        <div
          style={{
            background: "#f43f5e10",
            border: "1px solid #f43f5e30",
            borderRadius: "8px",
            padding: "16px 20px",
            color: "#fb7185",
            fontSize: "13px",
            fontFamily: "'JetBrains Mono', monospace",
          }}
        >
          {error}
        </div>
      ) : contracts.length === 0 ? (
        <div
          style={{
            textAlign: "center",
            padding: "60px 20px",
            color: "var(--text-muted)",
            fontSize: "13px",
          }}
        >
          <div style={{ fontSize: "32px", marginBottom: "12px", opacity: 0.3 }}>◈</div>
          No contracts registered yet.
          <div style={{ marginTop: "8px", fontSize: "12px" }}>
            Commit a contract YAML and open a PR to register one.
          </div>
        </div>
      ) : (
        <div
          style={{
            display: "grid",
            gridTemplateColumns: "repeat(auto-fill, minmax(340px, 1fr))",
            gap: "12px",
          }}
        >
          {contracts.map((contract) => (
            <ContractCard key={contract.id} contract={contract} />
          ))}
        </div>
      )}
    </div>
  );
}
