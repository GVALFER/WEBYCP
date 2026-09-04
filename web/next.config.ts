import type { NextConfig } from "next";

const apiUrl = (process.env.WEBYCP_API_URL || "http://127.0.0.1:8080").replace(/\/+$/, "");
const isDev = process.env.NODE_ENV !== "production";

const contentSecurityPolicy = [
    "default-src 'self'",
    "base-uri 'self'",
    `connect-src 'self'${isDev ? " ws: wss:" : ""}`,
    "font-src 'self' data:",
    "form-action 'self'",
    "frame-ancestors 'none'",
    "frame-src 'none'",
    "img-src 'self' data: blob:",
    "object-src 'none'",
    `script-src 'self' 'unsafe-inline'${isDev ? " 'unsafe-eval'" : ""}`,
    "script-src-attr 'none'",
    "style-src 'self' 'unsafe-inline'",
    "worker-src 'self' blob:",
    ...(isDev ? [] : ["upgrade-insecure-requests"]),
].join("; ");

const nextConfig: NextConfig = {
    output: "standalone",
    poweredByHeader: false,
    headers: async () => [
        {
            source: "/:path*",
            headers: [
                { key: "Content-Security-Policy", value: contentSecurityPolicy },
                {
                    key: "Permissions-Policy",
                    value: "camera=(), geolocation=(), microphone=()",
                },
                {
                    key: "Referrer-Policy",
                    value: "strict-origin-when-cross-origin",
                },
                { key: "X-Content-Type-Options", value: "nosniff" },
                { key: "X-Frame-Options", value: "DENY" },
                ...(!isDev
                    ? [
                          {
                              key: "Strict-Transport-Security",
                              value: "max-age=31536000",
                          },
                      ]
                    : []),
            ],
        },
    ],
    rewrites: async () => [
        {
            source: "/api/v1/:path*",
            destination: `${apiUrl}/api/v1/:path*`,
        },
    ],
};

export default nextConfig;
