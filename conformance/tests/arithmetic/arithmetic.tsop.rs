use tsop::prelude::*;

#[tsop::contract]
struct Arithmetic {
    #[readonly]
    target: Int,
}

#[tsop::methods(Arithmetic)]
impl Arithmetic {
    #[public]
    fn verify(&self, a: Int, b: Int) {
        let sum = a + b;
        let diff = a - b;
        let prod = a * b;
        let quot = a / b;
        let result = sum + diff + prod + quot;
        assert!(result == self.target);
    }
}
