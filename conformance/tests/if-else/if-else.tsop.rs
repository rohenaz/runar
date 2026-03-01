use tsop::prelude::*;

#[tsop::contract]
struct IfElse {
    #[readonly]
    limit: Int,
}

#[tsop::methods(IfElse)]
impl IfElse {
    #[public]
    fn check(&self, value: Int, mode: bool) {
        let mut result: Int = 0;
        if mode {
            result = value + self.limit;
        } else {
            result = value - self.limit;
        }
        assert!(result > 0);
    }
}
