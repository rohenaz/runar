import { describe, it } from 'vitest';
import { readFileSync } from 'fs';
import { resolve } from 'path';
import { parse, validate, typecheck, lowerToANF } from 'runar-compiler';
import type { ANFMethod } from 'runar-compiler';

const PROJECT_ROOT = resolve(import.meta.dirname, '..', '..', '..', '..');

describe('Inspect InductiveToken ANF', () => {
  it('dumps ANF for send method', () => {
    const source = readFileSync(
      resolve(PROJECT_ROOT, 'examples/ts/inductive-token/InductiveToken.runar.ts'),
      'utf-8',
    );
    const parseResult = parse(source, 'InductiveToken.runar.ts');
    const valResult = validate(parseResult.contract!);
    const tcResult = typecheck(valResult.contract ?? parseResult.contract!);
    const anfResult = lowerToANF(tcResult.contract ?? valResult.contract ?? parseResult.contract!);

    // Find the send method
    const sendMethod = anfResult.methods.find((m: ANFMethod) => m.name === 'send');
    if (!sendMethod) {
      console.log('Methods:', anfResult.methods.map((m: ANFMethod) => m.name));
      return;
    }

    console.log('=== send method ANF ===');
    console.log('params:', sendMethod.params.map((p: any) => `${p.name}:${p.type}`));
    console.log('bindings count:', sendMethod.body.length);

    // Print all bindings, focusing on if-else and extractOutpoint
    for (let i = 0; i < sendMethod.body.length; i++) {
      const b = sendMethod.body[i]!;
      const v = b.value;
      const kind = v.kind;
      if (kind === 'if') {
        console.log(`  [${i}] ${b.name} = if(${(v as any).condition})`);
        console.log(`    then: ${(v as any).thenBindings?.length ?? 0} bindings`);
        console.log(`    else: ${(v as any).elseBindings?.length ?? 0} bindings`);
        // Print first few bindings of each branch
        const thenB = (v as any).thenBindings || [];
        const elseB = (v as any).elseBindings || [];
        for (let j = 0; j < Math.min(5, thenB.length); j++) {
          console.log(`    then[${j}]: ${thenB[j].name} = ${thenB[j].value.kind}(${JSON.stringify(thenB[j].value).slice(0,100)})`);
        }
        if (thenB.length > 5) console.log(`    then[...${thenB.length - 5} more]`);
        for (let j = 0; j < Math.min(5, elseB.length); j++) {
          console.log(`    else[${j}]: ${elseB[j].name} = ${elseB[j].value.kind}(${JSON.stringify(elseB[j].value).slice(0,100)})`);
        }
        if (elseB.length > 5) console.log(`    else[...${elseB.length - 5} more]`);
      } else if (kind === 'call' && ((v as any).func === 'extractOutpoint' || (v as any).func?.includes('sha256'))) {
        console.log(`  [${i}] ${b.name} = ${kind}(${(v as any).func}, [${(v as any).args}])`);
      } else if (kind === 'load_param' || kind === 'load_prop') {
        console.log(`  [${i}] ${b.name} = ${kind}("${(v as any).name}")`);
      } else if (kind === 'assert') {
        console.log(`  [${i}] ${b.name} = assert(${(v as any).condition})`);
      } else if (kind === 'add_output') {
        console.log(`  [${i}] ${b.name} = add_output(${JSON.stringify(v).slice(0,150)})`);
      } else if (kind === 'check_sig') {
        console.log(`  [${i}] ${b.name} = check_sig(${(v as any).sig}, ${(v as any).pubKey})`);
      } else if (i < 50 || i > sendMethod.body.length - 20) {
        // Print first 50 and last 20
        const replacer = (_k: string, val: unknown) => typeof val === 'bigint' ? `${val}n` : val;
        console.log(`  [${i}] ${b.name} = ${kind}(${JSON.stringify(v, replacer).slice(0,120)})`);
      }
    }
  });
});
