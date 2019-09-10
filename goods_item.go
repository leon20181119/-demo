type GoodsItem struct {
	ID          int64
	CreatedAt   *time.Time
	UpdatedAt   *time.Time
	Amount      decimal.Decimal `sql:"type:decimal(20,4);"`
	MinAmount   decimal.Decimal `sql:"type:decimal(20,4);"`
	MaxAmount   decimal.Decimal `sql:"type:decimal(20,4);"`
	Url         string
	ProductName string
	BillType    string
}

type GoodsItemSlice []GoodsItem

func (a GoodsItemSlice) Len() int {
	return len(a)
}

func (a GoodsItemSlice) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a GoodsItemSlice) Less(i, j int) bool { // 重写 Less() 方法， 从大到小排序
	return a[j].Amount.LessThan(a[i].Amount)
}
