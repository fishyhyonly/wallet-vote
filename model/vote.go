package model

type VUser struct {
	UserId int64 `xorm:"pk"`
	UserPhone string
	UserEmail string
	UserOther string
	PowPrice float64
	LockPrice float64
	NodeId int64
	VoteArea string
	VoteNum float64
	TotalProfit float64
	SupNodeId int64
	SupVoteNum float64
	SupProfit float64
	VoteAddr string
	WalletId int64
	IsMac int
	VotePri string
	Version int `xorm:"version"` // 乐观锁
}

type UserVote struct {
	Id int64
	Area string
	Image string
	Name string
}

type VNodeInfo struct {
	Id int64
	Name string
	Ip string
	Area string
	Image string
	UserId int64
	ChainAddr string
	TotalAmount float64
	LockAmount float64
	MacAmount float64
	IssueNum int
	Status int //1进行中，2历史
	Stage int
	EndTime string
	CreateTime string
	Version int `xorm:"version"` // 乐观锁
}

type VMachine struct {
	Id int
	Addr string
	Ip string
}

type SWallet struct {
	Id int64
	EthAddr string   //kto回收地址
	EthPri string
	KtoPowAddr string   //kto原力分发地址
	KtoPowPri string
	KtoNodeAddr string   //kto节点分发地址
	KtoNodePri string
	KtoExcAddr string //kto兑换地址
	KtoExcPri string
	KtoMacAddr string  //kto矿机地址
	KtoCalAddr string  //kto矿机地址
	KtoLwAddr string  //kto矿机地址
	KtoMacPri string
	PowAmount float64
	PowCallAmount float64
	PowRatio float64
	MacRatio float64
	SupNum int
	/*PowNonce uint64
	NodeNonce uint64
	MacNonce uint64
	ExcNonce uint64*/
}

type VVoteBill struct {
	Id int64
	UserId int64
	UserPhone string
	UserEmail string
	UserOther string
	NodeName string
	NodeId int64
	NodeArea string
	NodeImage string
	Amount float64
	IssueNum int
	BillType int //1投票流水，2收益流水
	AmountType int //1投票流水，2收益流水，3其他
	CreateTime string
}

type VNodeDataBill struct {
	Id int
	TotalPl float64
	TotalVote float64
	KtoPrice float64
	MacAddrAmount float64
	PowAddrAmount float64
	NodeAddrAmount float64
	PowRatio float64
	UserNodeRatio float64
	MacNodeRatio float64
	TotalUserVote float64
	TotalMacVote float64
	CreateTime string
}

type WWalletTxhash struct {
	Id int64
	FromAddr string
	ToAddr string
	Txhash string
	Amount float64
	CreateTime string
}

type UserErrTx struct {
	UserId int64
	Addr string
	Amount float64
	ErrType int
	PowNum float64
	LockNum float64
}
