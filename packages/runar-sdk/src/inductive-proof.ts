/**
 * InductiveProofManager — manages ZKP proof lifecycle for InductiveSmartContract.
 *
 * The proof manager handles:
 * 1. Storing the current proof (from the previous spend)
 * 2. Generating a new proof for the next spend (delegated to an external prover)
 * 3. Encoding proofs for on-chain verification
 *
 * The on-chain `snark_verify` checks the proof against the genesis outpoint.
 * The proof attests that the entire chain from genesis to the current transaction
 * is valid.
 *
 * Current implementation: MOCK — uses zero-filled proofs that pass the OP_TRUE stub.
 * Real implementation will use Groth16 proofs from `runar-zkp`.
 */

/** Size of the proof field in bytes. */
export const PROOF_SIZE = 192;

/** Zero proof (passes the OP_TRUE stub verifier). */
export const ZERO_PROOF = '00'.repeat(PROOF_SIZE);

/**
 * Proof generation function signature.
 * Takes genesis outpoint and previous state, returns a proof hex string.
 */
export type ProofGenerator = (
  genesisOutpoint: string,
  previousProof: string,
  parentTxId: string,
  parentState: Record<string, unknown>,
) => Promise<string>;

/**
 * Manages proof state for an inductive contract instance.
 */
export class InductiveProofManager {
  private _proof: string;
  private _generator: ProofGenerator | null;

  constructor(initialProof?: string, generator?: ProofGenerator) {
    this._proof = initialProof ?? ZERO_PROOF;
    this._generator = generator ?? null;
  }

  /** Get the current proof hex string. */
  get proof(): string {
    return this._proof;
  }

  /** Set the proof (e.g., after deserializing from chain state). */
  set proof(value: string) {
    if (value.length !== PROOF_SIZE * 2) {
      throw new Error(
        `InductiveProofManager: proof must be ${PROOF_SIZE} bytes (${PROOF_SIZE * 2} hex chars), got ${value.length / 2} bytes`,
      );
    }
    this._proof = value;
  }

  /**
   * Generate a new proof for the next spend.
   *
   * If no generator is configured, returns the zero proof (for stub verifier).
   * With a real prover, this would compute a Groth16 proof.
   */
  async generateProof(
    genesisOutpoint: string,
    parentTxId: string,
    parentState: Record<string, unknown>,
  ): Promise<string> {
    if (this._generator) {
      const newProof = await this._generator(
        genesisOutpoint,
        this._proof,
        parentTxId,
        parentState,
      );
      this._proof = newProof;
      return newProof;
    }
    // No generator — return zero proof (stub verifier accepts anything)
    return ZERO_PROOF;
  }

  /**
   * Check if this manager has a real proof generator.
   */
  get hasGenerator(): boolean {
    return this._generator !== null;
  }
}
