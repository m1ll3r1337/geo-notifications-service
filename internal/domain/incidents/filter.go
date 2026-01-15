package incidents

type ListFilter struct {
	Limit      int
	Offset     int
	ActiveOnly bool
}
