import { defineConfig } from '@playwright/test';

export default defineConfig({
    baseURL: 'http://localhost:8080',
    fullyParallel: false,
    workers: 1,
    retries: 0,
    use: {
        trace: 'on-first-retry',
    },
    projects: [
        {
            name: 'setup',
            testMatch: /.*\.setup\.ts/,
        },
        {
            name: 'member-tests',
            testMatch: /^(?!.*admin-).*\.spec\.ts$/,
            dependencies: ['setup'],
            use: { storageState: '.auth/member.json' },
        },
        {
            name: 'admin-tests',
            testMatch: /admin-.*\.spec\.ts$/,
            dependencies: ['setup'],
            use: { storageState: '.auth/admin.json' },
        },
    ],
});
