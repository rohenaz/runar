/**
 * Smoke test: compiles InductiveToken through the full pipeline.
 */
import { describe, it, expect } from 'vitest';
import { readFileSync } from 'fs';
import { resolve } from 'path';
import { parse, validate, typecheck, lowerToANF } from '../index.js';
import { lowerToStack } from '../passes/05-stack-lower.js';

const PROJECT_ROOT = resolve(import.meta.dirname, '..', '..', '..', '..');

describe('InductiveToken stack compilation', () => {
  it('compiles InductiveToken through full pipeline without errors', () => {
    const source = readFileSync(
      resolve(PROJECT_ROOT, 'examples/ts/inductive-token/InductiveToken.runar.ts'),
      'utf-8',
    );
    const parseResult = parse(source, 'InductiveToken.runar.ts');
    const contract = parseResult.contract!;
    validate(contract);
    typecheck(contract);
    const anf = lowerToANF(contract);
    const stackProgram = lowerToStack(anf);

    const sendMethod = stackProgram.methods.find((m) => m.name === 'send');
    expect(sendMethod).toBeTruthy();
    expect(sendMethod!.ops.length).toBeGreaterThan(0);

    const transferMethod = stackProgram.methods.find((m) => m.name === 'transfer');
    expect(transferMethod).toBeTruthy();
    expect(transferMethod!.ops.length).toBeGreaterThan(0);
  });
});
