import { describe, it, expect } from 'vitest';
import { parse } from '../passes/01-parse.js';
import { lowerToANF } from '../passes/04-anf-lower.js';

describe('ecMul debug 4', () => {
  it('compare ANF: param vs constant', () => {
    const contract1 = `
import { SmartContract, assert, ecMul, ecPointX } from 'runar-lang';
import type { Point } from 'runar-lang';

class Test extends SmartContract {
  readonly pt: Point;
  constructor(pt: Point) {
    super(pt);
    this.pt = pt;
  }
  public test1(scalar: bigint) {
    const result = ecMul(this.pt, scalar);
    assert(ecPointX(result) === ecPointX(this.pt));
  }
}
`;

    const contract2 = `
import { SmartContract, assert, ecMul, ecPointX } from 'runar-lang';
import type { Point } from 'runar-lang';

class Test extends SmartContract {
  readonly pt: Point;
  constructor(pt: Point) {
    super(pt);
    this.pt = pt;
  }
  public test2() {
    const result = ecMul(this.pt, 1n);
    assert(ecPointX(result) === ecPointX(this.pt));
  }
}
`;

    const parse1 = parse(contract1, 'Test.runar.ts');
    const parse2 = parse(contract2, 'Test.runar.ts');
    
    const anf1 = lowerToANF(parse1.contract!);
    const anf2 = lowerToANF(parse2.contract!);
    
    const m1 = anf1.methods.find(m => m.name === 'test1');
    const m2 = anf2.methods.find(m => m.name === 'test2');

    console.log('\n=== test1 (with scalar param) - first 5 bindings ===');
    if (m1) {
      m1.body.slice(0, 5).forEach((b: any) => {
        console.log(`${b.name}: kind=${b.value.kind}, func=${b.value.func || 'N/A'}`);
      });
    }

    console.log('\n=== test2 (with constant 1n) - first 5 bindings ===');
    if (m2) {
      m2.body.slice(0, 5).forEach((b: any) => {
        console.log(`${b.name}: kind=${b.value.kind}, func=${b.value.func || 'N/A'}`);
      });
    }

    console.log('\n=== test2 FULL bindings ===');
    if (m2) {
      m2.body.forEach((b: any) => {
        console.log(`${b.name}: kind=${b.value.kind}, func=${b.value.func || 'N/A'}`);
      });
    }

    expect(m1?.body.length).toBeGreaterThan(0);
    expect(m2?.body.length).toBeGreaterThan(0);
  });
});
