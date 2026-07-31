// Registers the jest-dom matchers (toBeInTheDocument, toBeDisabled, ...) with
// vitest's expect, and their type augmentation for tsc.
import '@testing-library/jest-dom/vitest';

import { cleanup } from '@testing-library/react';
import { afterEach } from 'vitest';

// Testing Library only auto-cleans when the test framework exposes a global
// afterEach. Tests import from vitest explicitly rather than relying on
// globals, so unmounting has to be wired up here — otherwise every render
// accumulates in the document and queries match elements from earlier tests.
afterEach(cleanup);
