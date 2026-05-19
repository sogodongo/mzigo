/**
 * Server-side API client for the catalog BFF layer.
 *
 * All functions here run in Next.js API routes or Server Components.
 * They are never bundled into the browser. Service URLs and any auth
 * headers live only on the server.
 *
 * Error handling: functions throw on non-OK responses. API route handlers
 * catch these and return appropriate HTTP status codes to the browser.
 */

import type {
  BlastRadiusReport,
  Contract,
  ContractVersion,
  FieldPolicy,
  ApprovalEvent,
} from "./types";

const CONTRACTS_URL = process.env.CONTRACTS_SERVICE_URL!;
const ANALYZER_URL = process.env.ANALYZER_SERVICE_URL!;

async function fetchJSON<T>(url: string, init?: RequestInit): Promise<T> {
  const res = await fetch(url, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...init?.headers,
    },
    // Next.js fetch caching: revalidate every 30s.
    // Contract data changes infrequently; a short cache avoids hammering
    // the contracts service on every page render.
    next: { revalidate: 30 },
  });

  if (!res.ok) {
    throw new Error(`API error ${res.status} from ${url}`);
  }

  return res.json() as Promise<T>;
}

export async function listContracts(): Promise<Contract[]> {
  return fetchJSON<Contract[]>(`${CONTRACTS_URL}/v1/contracts`);
}

export async function getContract(name: string): Promise<Contract> {
  return fetchJSON<Contract>(`${CONTRACTS_URL}/v1/contracts/${encodeURIComponent(name)}`);
}

export async function getContractVersions(contractId: string): Promise<ContractVersion[]> {
  return fetchJSON<ContractVersion[]>(
    `${CONTRACTS_URL}/v1/contracts/${contractId}/versions`
  );
}

export async function getFieldPolicies(contractId: string): Promise<FieldPolicy[]> {
  return fetchJSON<FieldPolicy[]>(
    `${CONTRACTS_URL}/v1/contracts/${contractId}/field-policies`
  );
}

export async function getApprovalHistory(contractId: string): Promise<ApprovalEvent[]> {
  return fetchJSON<ApprovalEvent[]>(
    `${CONTRACTS_URL}/v1/contracts/${contractId}/approvals`
  );
}

export async function computeBlastRadius(
  topic: string,
  changedFields: string[]
): Promise<BlastRadiusReport> {
  return fetchJSON<BlastRadiusReport>(`${ANALYZER_URL}/v1/blast-radius`, {
    method: "POST",
    body: JSON.stringify({ topic, changed_fields: changedFields }),
    // Blast-radius analysis is expensive. Do not cache: always compute fresh.
    next: { revalidate: 0 },
  });
}
