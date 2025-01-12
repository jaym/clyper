import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  async rewrites() {
    return [
      {
        source: '/api/:path*',
        destination: 'http://localhost:8991/:path*', // Replace with your API server's port
      },
    ];
  },
};

export default nextConfig;
