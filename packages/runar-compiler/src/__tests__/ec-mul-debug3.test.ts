import { describe, it } from 'vitest';
import { parse } from '../passes/01-parse.js';

describe('ecMul debug 3', () => {
  it('show AST for constant scalar', () => {
    const contract = `
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

    const parseResult = parse(contract, 'Test.runar.ts');
    const ast = parseResult.contract!;
    const method = ast.methods.find((m: any) => m.name === 'test2');
    
    console.log('\n=== test2 AST ===');
    if (method && method.body) {
      method.body.forEach((stmt: any, i: number) => {
        console.log(`\n${i}: ${stmt.kind}`);
        if (stmt.kind === 'var_declaration') {
          console.log(`   name: ${stmt.name}`);
          if (stmt.init && stmt.init.kind === 'call_expr') {
            console.log(`   init call_expr:`);
            console.log(`     callee.name: ${stmt.init.callee.name}`);
            console.log(`     args length: ${stmt.init.args.length}`);
            console.log(`     args[0].kind: ${stmt.init.args[0].kind}`);
            console.log(`     args[1].kind: ${stmt.init.args[1].kind}`);
            if (stmt.init.args[1].kind === 'bigint_literal') {
              console.log(`     args[1].value: ${stmt.init.args[1].value}`);
            }
          }
        }
      });
    }
  });
});
