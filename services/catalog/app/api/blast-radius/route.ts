import { NextRequest, NextResponse } from "next/server";
import { computeBlastRadius } from "@/lib/api";

// POST /api/blast-radius
// Proxies the browser's blast-radius request to the analyzer service.
// Keeping this in an API route means the analyzer service URL and any
// internal auth tokens never reach the browser.
export async function POST(req: NextRequest) {
  try {
    const body = await req.json();
    const { topic, changed_fields } = body;

    if (!topic || !Array.isArray(changed_fields) || changed_fields.length === 0) {
      return NextResponse.json(
        { error: "topic and changed_fields are required" },
        { status: 400 }
      );
    }

    const report = await computeBlastRadius(topic, changed_fields);
    return NextResponse.json(report);
  } catch (e) {
    const msg = e instanceof Error ? e.message : "analysis failed";
    return NextResponse.json({ error: msg }, { status: 502 });
  }
}
