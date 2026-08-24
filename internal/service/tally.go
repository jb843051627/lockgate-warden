package service

// tally 质量计数器的服务层薄封装，供 KPI 与评估共用。
type tally struct {
	good     int64
	suspect  int64
	rejected int64
}

func (t *tally) add(good, suspect, rejected int64) {
	t.good += good
	t.suspect += suspect
	t.rejected += rejected
}

func (t tally) total() int64 {
	return t.good + t.suspect + t.rejected
}

// rate 优良率：无数据窗口视为全优（返回 1）。
func (t tally) rate() float64 {
	total := t.total()
	return float64(t.good) / float64(total)
}
