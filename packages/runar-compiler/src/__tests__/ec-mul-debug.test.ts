import { describe, it } from 'vitest';
import { compile } from '../index.js';

describe('ecMul debug', () => {
  it('compare ANF for param vs constant scalar', () => {
    const PT_HEX = '79be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798' +
                   '483ada7726a3c4655da4fbfc0e1108a8fd17b448a68554199c47d08ffb10d4b8';

    const contract = `
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
  public test2() {
    const result = ecMul(this.pt, 1n);
    assert(ecPointX(result) === ecPointX(this.pt));
  }
}
`;

    const result = compile(contract, {
      fileName: 'Test.runar.ts',
      constructorArgs: { pt: PT_HEX },
    });

    if (result.anf) {
      const m1 = result.anf.methods.find((m: any) => m.name === 'test1');
      const m2 = result.anf.methods.find((m: any) => m.name === 'test2');
      
      console.log('\n=== test1 (param) ANF ===');
      if (m1 && m1.body) {
        m1.body.slice(0, 5).forEach((b: any) => {
          console.log(`${b.name}: ${JSON.stringify(b.value)}`);
        });
      }

      console.log('\n=== test2 (constant) ANF ===');
      if (m2 && m2.body) {
        m2.body.slice(0, 5).forEach((b: any) => {
          console.log(`${b.name}: ${JSON.stringify(b.value)}`);
        });
      }
    }
  });
});
