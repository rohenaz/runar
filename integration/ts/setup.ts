/**
 * Vitest globalSetup — checks node availability and mines initial blocks.
 */

import { isNodeAvailable, getBlockCount, mine } from './helpers/node.js';

export default async function setup() {
  const available = await isNodeAvailable();
  if (!available) {
    console.error('Regtest node not running. Skipping integration tests.');
    console.error('Start with: cd integration && ./regtest.sh start');
    process.exit(0);
  }

  const height = await getBlockCount();
  const target = 101;
  const needed = target - height;
  if (needed > 0) {
    console.error(`Mining ${needed} blocks (current: ${height}, target: ${target})...`);
    await mine(needed);
  }
}
