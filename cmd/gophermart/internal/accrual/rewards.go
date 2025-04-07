package accrual

var Rewards = []Reward{
	{"Samsung", 10, "%"},
	{"Apple", 5, "%"},
	{"Xiaomi", 20, "%"},
}

type Reward struct {
	Match      string  `json:"match"`
	Reward     float64 `json:"reward"`
	RewardType string  `json:"reward_type"`
}

type Order struct {
	Number string `json:"order"`
	Goods  []Good `json:"goods"`
}
type Good struct {
	Description string  `json:"description"`
	Price       float64 `json:"price"`
}
