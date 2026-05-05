import path from "node:path";
import { fileURLToPath } from "node:url";

import type { NextConfig } from "next";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// API_INTERNAL_URL is used at build time for Next.js server-side rewrites.
// In Docker it points to the internal service name; locally falls back to localhost.
const API_BASE =
  process.env.API_INTERNAL_URL ??
  process.env.NEXT_PUBLIC_API_BASE_URL ??
  "http://localhost:8080";

const nextConfig: NextConfig = {
  reactStrictMode: true,
  output: "standalone",
  outputFileTracingRoot: path.join(__dirname, ".."),
  async rewrites() {
    return [
      {
        source: "/v1/:path*",
        destination: `${API_BASE}/v1/:path*`,
      },
    ];
  },
};

export default nextConfig;
