import { defineConfig } from "vite";
import react from "@vitejs/plugin-react-swc";

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      "/query": "http://localhost:8080",
      "/download": "http://localhost:8080",
    },
  },
});
