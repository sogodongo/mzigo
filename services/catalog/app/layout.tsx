import type { Metadata } from "next";
import Link from "next/link";
import "./globals.css";

export const metadata: Metadata = {
  title: "Mzigo Catalog",
  description: "Streaming data contracts, lineage, and governance",
};

const NAV = [
  { href: "/contracts",    label: "Contracts",    icon: "◈" },
  { href: "/lineage",      label: "Lineage",      icon: "⬡" },
  { href: "/blast-radius", label: "Blast Radius", icon: "◎" },
];

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body>
        <div className="flex min-h-screen">
          <aside
            style={{
              width: "220px",
              flexShrink: 0,
              background: "var(--bg-surface)",
              borderRight: "1px solid var(--border)",
              display: "flex",
              flexDirection: "column",
              padding: "0",
              position: "fixed",
              top: 0,
              left: 0,
              height: "100vh",
              zIndex: 40,
            }}
          >
            {/* Wordmark */}
            <div
              style={{
                padding: "24px 20px 20px",
                borderBottom: "1px solid var(--border)",
              }}
            >
              <Link href="/" style={{ textDecoration: "none" }}>
                <span
                  style={{
                    fontFamily: "'JetBrains Mono', monospace",
                    fontWeight: 600,
                    fontSize: "18px",
                    color: "var(--accent)",
                    letterSpacing: "-0.5px",
                  }}
                >
                  mzigo
                </span>
                <span
                  style={{
                    display: "block",
                    fontSize: "10px",
                    color: "var(--text-muted)",
                    fontFamily: "'JetBrains Mono', monospace",
                    letterSpacing: "0.08em",
                    marginTop: "2px",
                  }}
                >
                  CONTROL PLANE
                </span>
              </Link>
            </div>

            {/* Nav links */}
            <nav style={{ padding: "12px 8px", flex: 1 }}>
              {NAV.map(({ href, label, icon }) => (
                <Link
                  key={href}
                  href={href}
                  style={{
                    display: "flex",
                    alignItems: "center",
                    gap: "10px",
                    padding: "9px 12px",
                    borderRadius: "6px",
                    color: "var(--text-secondary)",
                    textDecoration: "none",
                    fontSize: "13px",
                    fontWeight: 500,
                    marginBottom: "2px",
                    transition: "background 0.15s, color 0.15s",
                  }}
                  className="nav-link"
                >
                  <span
                    style={{
                      fontFamily: "monospace",
                      fontSize: "14px",
                      color: "var(--text-muted)",
                      width: "16px",
                      textAlign: "center",
                    }}
                  >
                    {icon}
                  </span>
                  {label}
                </Link>
              ))}
            </nav>

            {/* Footer */}
            <div
              style={{
                padding: "16px 20px",
                borderTop: "1px solid var(--border)",
                fontSize: "11px",
                color: "var(--text-muted)",
                fontFamily: "'JetBrains Mono', monospace",
              }}
            >
              v0.1.0
            </div>
          </aside>

          {/* Main content */}
          <main
            style={{
              marginLeft: "220px",
              flex: 1,
              minHeight: "100vh",
              background: "var(--bg-base)",
            }}
          >
            {children}
          </main>
        </div>

        <style>{`
          .nav-link:hover {
            background: var(--bg-raised);
            color: var(--text-primary);
          }
        `}</style>
      </body>
    </html>
  );
}
