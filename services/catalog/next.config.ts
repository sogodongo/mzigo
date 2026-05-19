import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // Strict mode surfaces potential issues during development.
  reactStrictMode: true,

  // Environment variables available server-side only.
  // These are injected at runtime in Kubernetes via ConfigMap/Secret.
  // They are NOT exposed to the browser; all backend calls go through
  // the API routes (BFF layer).
  env: {
    CONTRACTS_SERVICE_URL: process.env.CONTRACTS_SERVICE_URL ?? "http://contracts:8081",
    ANALYZER_SERVICE_URL: process.env.ANALYZER_SERVICE_URL ?? "http://analyzer:8083",
    LINEAGE_SERVICE_URL: process.env.LINEAGE_SERVICE_URL ?? "http://marquez:5000",
  },
};

export default nextConfig;
