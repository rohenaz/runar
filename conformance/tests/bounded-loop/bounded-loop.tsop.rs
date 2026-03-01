use tsop::prelude::*;

#[tsop::contract]
struct BoundedLoop {
    #[readonly]
    expected_sum: Int,
}

#[tsop::methods(BoundedLoop)]
impl BoundedLoop {
    #[public]
    fn verify(&self, start: Int) {
        let mut sum: Int = 0;
        for i in 0..5 {
            sum = sum + start + i;
        }
        assert!(sum == self.expected_sum);
    }
}
