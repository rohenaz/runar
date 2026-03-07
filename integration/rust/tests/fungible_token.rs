//! FungibleToken integration test — stateful contract with addOutput (SDK Deploy path).
//!
//! All methods require a Sig parameter (checkSig), so spending requires raw
//! transaction construction. We test compile + deploy via the SDK.

use crate::helpers::*;
use runar_lang::sdk::{DeployOptions, RunarContract, SdkValue};

fn hex_encode_str(s: &str) -> String {
    s.as_bytes().iter().map(|b| format!("{:02x}", b)).collect()
}

#[test]
#[ignore]
fn test_fungible_token_compile() {
    skip_if_no_node();

    let artifact = compile_contract("examples/ts/token-ft/FungibleTokenExample.runar.ts");
    assert_eq!(artifact.contract_name, "FungibleToken");
}

#[test]
#[ignore]
fn test_fungible_token_deploy() {
    skip_if_no_node();

    let artifact = compile_contract("examples/ts/token-ft/FungibleTokenExample.runar.ts");

    let mut provider = create_provider();
    let owner = create_wallet();
    let (signer, _wallet) = create_funded_wallet(&mut provider);

    let token_id_hex = hex_encode_str("TEST-TOKEN-001");

    // Constructor: (owner: PubKey, balance: bigint, tokenId: ByteString)
    let mut contract = RunarContract::new(artifact, vec![
        SdkValue::Bytes(owner.pub_key_hex),
        SdkValue::Int(1000),
        SdkValue::Bytes(token_id_hex),
    ]);

    let (deploy_txid, _tx) = contract
        .deploy(&mut provider, &*signer, &DeployOptions {
            satoshis: 5000,
            change_address: None,
        })
        .expect("deploy failed");
    assert!(!deploy_txid.is_empty());
    assert_eq!(deploy_txid.len(), 64);
}

#[test]
#[ignore]
fn test_fungible_token_deploy_zero_balance() {
    skip_if_no_node();

    let artifact = compile_contract("examples/ts/token-ft/FungibleTokenExample.runar.ts");

    let mut provider = create_provider();
    let owner = create_wallet();
    let (signer, _wallet) = create_funded_wallet(&mut provider);

    let token_id_hex = hex_encode_str("ZERO-BAL-TOKEN");

    let mut contract = RunarContract::new(artifact, vec![
        SdkValue::Bytes(owner.pub_key_hex),
        SdkValue::Int(0),
        SdkValue::Bytes(token_id_hex),
    ]);

    let (deploy_txid, _tx) = contract
        .deploy(&mut provider, &*signer, &DeployOptions {
            satoshis: 5000,
            change_address: None,
        })
        .expect("deploy failed");
    assert!(!deploy_txid.is_empty());
}

#[test]
#[ignore]
fn test_fungible_token_deploy_large_balance() {
    skip_if_no_node();

    let artifact = compile_contract("examples/ts/token-ft/FungibleTokenExample.runar.ts");

    let mut provider = create_provider();
    let owner = create_wallet();
    let (signer, _wallet) = create_funded_wallet(&mut provider);

    let token_id_hex = hex_encode_str("BIG-TOKEN");

    // Note: SdkValue::Int is i64, so max is ~9.2 * 10^18
    let mut contract = RunarContract::new(artifact, vec![
        SdkValue::Bytes(owner.pub_key_hex),
        SdkValue::Int(2_100_000_000_000_000),
        SdkValue::Bytes(token_id_hex),
    ]);

    let (deploy_txid, _tx) = contract
        .deploy(&mut provider, &*signer, &DeployOptions {
            satoshis: 5000,
            change_address: None,
        })
        .expect("deploy failed");
    assert!(!deploy_txid.is_empty());
}

#[test]
#[ignore]
fn test_fungible_token_send() {
    skip_if_no_node();

    let artifact = compile_contract("examples/ts/token-ft/FungibleTokenExample.runar.ts");
    let mut provider = create_provider();
    let (signer, owner_wallet) = create_funded_wallet(&mut provider);
    let recipient = create_wallet();

    let token_id_hex = "deadbeef".to_string();

    let mut contract = RunarContract::new(artifact, vec![
        SdkValue::Bytes(owner_wallet.pub_key_hex.clone()),
        SdkValue::Int(1000),
        SdkValue::Bytes(token_id_hex),
    ]);

    contract
        .deploy(&mut provider, &*signer, &DeployOptions {
            satoshis: 5000,
            change_address: None,
        })
        .expect("deploy failed");

    // send uses addOutput: the on-chain script expects output with owner=to, balance=1000
    // Pass new_state so the SDK builds the correct continuation output
    let recipient_pub = recipient.pub_key_hex.clone();
    let mut new_state = std::collections::HashMap::new();
    new_state.insert("owner".to_string(), SdkValue::Bytes(recipient.pub_key_hex));
    let call_opts = runar_lang::sdk::CallOptions {
        satoshis: None,
        change_address: None,
        new_state: Some(new_state),
    };
    let (call_txid, _tx) = contract
        .call(
            "send",
            &[SdkValue::Auto, SdkValue::Bytes(recipient_pub), SdkValue::Int(5000)],
            &mut provider,
            &*signer,
            Some(&call_opts),
        )
        .expect("send failed");
    assert!(!call_txid.is_empty());
}

#[test]
#[ignore]
fn test_fungible_token_wrong_owner_rejected() {
    skip_if_no_node();

    let artifact = compile_contract("examples/ts/token-ft/FungibleTokenExample.runar.ts");
    let mut provider = create_provider();
    // Deploy with owner=walletA
    let (signer_a, owner_wallet) = create_funded_wallet(&mut provider);
    let recipient = create_wallet();

    let token_id_hex = hex_encode_str("REJECT-TOKEN");

    let mut contract = RunarContract::new(artifact, vec![
        SdkValue::Bytes(owner_wallet.pub_key_hex.clone()),
        SdkValue::Int(1000),
        SdkValue::Bytes(token_id_hex),
    ]);

    contract
        .deploy(&mut provider, &*signer_a, &DeployOptions {
            satoshis: 5000,
            change_address: None,
        })
        .expect("deploy failed");

    // Call send with a different signer — should be rejected
    let (signer_b, _wallet_b) = create_funded_wallet(&mut provider);
    let mut new_state = std::collections::HashMap::new();
    new_state.insert("owner".to_string(), SdkValue::Bytes(recipient.pub_key_hex.clone()));
    let call_opts = runar_lang::sdk::CallOptions {
        satoshis: None,
        change_address: None,
        new_state: Some(new_state),
    };
    let result = contract.call(
        "send",
        &[SdkValue::Auto, SdkValue::Bytes(recipient.pub_key_hex), SdkValue::Int(5000)],
        &mut provider,
        &*signer_b,
        Some(&call_opts),
    );
    assert!(result.is_err(), "send with wrong owner should be rejected");
}
