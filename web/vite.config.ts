import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), "");
  const apiTarget = env.VITE_JANUS_API_BASE_URL || "http://localhost:8080";

  return {
    plugins: [react()],
    server: {
      port: 5173,
      proxy: {
        "/v1": {
          target: apiTarget,
          changeOrigin: true
        },
        "/ping": {
          target: apiTarget,
          changeOrigin: true
        }
      }
    }
  };
});
