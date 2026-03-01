use tsop::prelude::*;

#[tsop::contract]
pub struct CovenantVault {
    #[readonly]
    pub owner: PubKey,
    #[readonly]
    pub recipient: Addr,
    #[readonly]
    pub min_amount: Bigint,
}

#[tsop::methods(CovenantVault)]
impl CovenantVault {
    #[public]
    pub fn spend(&self, sig: &Sig, amount: Bigint, tx_preimage: &SigHashPreimage) {
        assert!(check_sig(sig, &self.owner));
        assert!(check_preimage(tx_preimage));
        assert!(amount >= self.min_amount);
    }
}
