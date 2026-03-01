package contract

import (
	"testing"
	"tsop"
)

func newAuction() *Auction {
	return &Auction{
		Auctioneer:    tsop.MockPubKey(),
		HighestBidder: tsop.PubKey("initial_bidder_placeholder_33b!"),
		HighestBid:    100,
		Deadline:      1000,
	}
}

func TestAuction_Bid(t *testing.T) {
	c := newAuction()
	bidder := tsop.PubKey("new_bidder_placeholder_33bytes!")
	c.Bid(bidder, 200)
	if c.HighestBid != 200 {
		t.Errorf("expected HighestBid=200, got %d", c.HighestBid)
	}
}

func TestAuction_Bid_MustBeHigher(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected assertion failure")
		}
	}()
	newAuction().Bid(tsop.MockPubKey(), 50)
}

func TestAuction_MultipleBids(t *testing.T) {
	c := newAuction()
	c.Bid(tsop.PubKey("bidder1_33bytes_placeholder_____"), 200)
	c.Bid(tsop.PubKey("bidder2_33bytes_placeholder_____"), 300)
	if c.HighestBid != 300 {
		t.Errorf("expected HighestBid=300, got %d", c.HighestBid)
	}
}

func TestAuction_Close(t *testing.T) {
	c := newAuction()
	c.Deadline = 0
	c.Close(tsop.MockSig())
}

func TestAuction_Close_BeforeDeadline_Fails(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected assertion failure")
		}
	}()
	newAuction().Close(tsop.MockSig())
}

func TestAuction_Compile(t *testing.T) {
	if err := tsop.CompileCheck("Auction.tsop.go"); err != nil {
		t.Fatalf("TSOP compile check failed: %v", err)
	}
}
