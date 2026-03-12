/**
 * IVC (Incrementally Verifiable Computation) step circuit.
 *
 * Defines the recursive circuit for chain verification:
 * - Public inputs: genesisOutpoint, parentStateHash, parentTxId
 * - Private inputs: previous proof, parent state
 * - Constraint: verify previous proof OR base case (genesis)
 *
 * This is a placeholder — actual circuit definition requires a
 * constraint system (R1CS) and a proving system (Nova/Groth16).
 */

export interface IVCPublicInputs {
  genesisOutpoint: Uint8Array; // 36 bytes
  parentStateHash: Uint8Array; // 32 bytes (SHA-256 of parent state)
  parentTxId: Uint8Array;      // 32 bytes
}

export interface IVCWitness {
  previousProof: Uint8Array;   // proof from parent transaction
  parentState: Uint8Array;     // raw state bytes from parent
  isGenesis: boolean;          // true for base case
}

/**
 * Define the IVC step circuit constraints.
 *
 * Returns a description of the R1CS constraints that will be compiled
 * into a Groth16 circuit. Placeholder for future implementation.
 */
export function defineIVCStepCircuit(): {
  numConstraints: number;
  numPublicInputs: number;
  numPrivateInputs: number;
} {
  return {
    // Estimated constraint count for the IVC step:
    // - SHA-256 hash verification: ~25,000 constraints
    // - Previous proof verification (recursive): ~10,000 constraints
    // - Genesis detection: ~100 constraints
    // - State hash computation: ~25,000 constraints
    numConstraints: 60_000,
    numPublicInputs: 3, // genesis, parentStateHash, parentTxId
    numPrivateInputs: 2, // previousProof, parentState
  };
}
