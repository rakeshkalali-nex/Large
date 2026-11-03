import { defineConfig } from 'vitest/config';
import angular from '@analogjs/vite-plugin-angular';
export default defineConfig({
  plugins: [angular({ tsconfig: './tsconfig.spec.json' })],
  test: {
    environment: 'jsdom',
    // Override Analog's vmThreads default so each suite has an isolated
    // Node module registry as well as its own browser globals.
    pool: 'forks',
    isolate: true,
    setupFiles: ['src/app/setup/test-setup.ts'],
    // Keep runtime and testing bundles in the same module graph: Angular's
    // platform registry is module-local and must not be split across loaders.
    server: { deps: { inline: [/@angular\//] } },
    include: ['src/**/*.spec.ts'],
    watch: false
  }
});
