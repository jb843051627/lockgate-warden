package model

// LockStatus 闸室运行等级（评估合成结果，同步到闸室档案）。
type LockStatus string

// 四档：正常 / 受限 / 停航 / 检修。
const (
	LockOpen        LockStatus = "open"
	LockRestricted  LockStatus = "restricted"
	LockClosed      LockStatus = "closed"
	LockMaintenance LockStatus = "maintenance"
)

// GateStatus 闸门行程状态机：Sealed↔Opening→Open↔Closing；Fault 需人工复位。
type GateStatus string

// 闸门状态集合。
const (
	GateSealed  GateStatus = "sealed"
	GateOpening GateStatus = "opening"
	GateOpen    GateStatus = "open"
	GateClosing GateStatus = "closing"
	GateFault   GateStatus = "fault"
)

// gateTransitions 合法后继转移表（顺序敏感，仅供 AllowedTransitions 复制）。
var gateTransitions = map[GateStatus][]GateStatus{
	GateSealed:  {GateOpening, GateFault},
	GateOpening: {GateOpen, GateFault},
	GateOpen:    {GateClosing, GateFault},
	GateClosing: {GateSealed, GateFault},
	GateFault:   {GateSealed},
}

// AllowedTransitions 返回当前闸门状态的合法后继集合。
func AllowedTransitions(from GateStatus) []GateStatus {
	return gateTransitions[from]
}

// IsValidTransition 判断一次状态流转是否合法。
func IsValidTransition(from, to GateStatus) bool {
	for _, next := range AllowedTransitions(from) {
		if next == to {
			return true
		}
	}
	return false
}

// SensorKind 传感器种类。
type SensorKind string

// level=水位计 position=闸位计 wind=风速 flow=流速计。
const (
	KindLevel SensorKind = "level"
	KindPos   SensorKind = "position"
	KindWind  SensorKind = "wind"
	KindFlow  SensorKind = "flow"
)

// Quality 遥测点三级质量结论。
type Quality string

// good / suspect / rejected。
const (
	QualityGood     Quality = "good"
	QualitySuspect  Quality = "suspect"
	QualityRejected Quality = "rejected"
)
