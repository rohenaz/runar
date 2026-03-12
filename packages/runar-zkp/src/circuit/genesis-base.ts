/**
 * Genesis base case circuit.
 *
 * For the first transaction in a chain (genesis), no previous proof exists.
 * The base case circuit proves that the genesis outpoint matches the
 * current transaction's outpoint.
 *
 * This is a placeholder — actual circuit requires R1CS compilation.
 */

export interface GenesisPublicInputs {
  genesisOutpoint: Uint8Array; // 36 bytes — the outpoint being spent
}

export function defineGenesisCircuit(): {
  numConstraints: number;
  numPublicInputs: number;
} {
  return {
    numConstraints: 100, // trivial circuit
    numPublicInputs: 1,  // genesisOutpoint
  };
}
