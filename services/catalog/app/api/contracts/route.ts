import { NextResponse } from "next/server";
import { listContracts } from "@/lib/api";

export async function GET() {
  try {
    const contracts = await listContracts();
    return NextResponse.json(contracts);
  } catch (e) {
    const msg = e instanceof Error ? e.message : "failed to fetch contracts";
    return NextResponse.json({ error: msg }, { status: 502 });
  }
}
