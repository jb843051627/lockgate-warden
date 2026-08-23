package model

import (
	"errors"
	"time"
)

// Gate 闸门档案：热胀冷缩修正的净空与错位上限。
type Gate struct {
	ID               int64      `json:"id"`
	Code             string     `json:"code"`
	ChamberID        int64      `json:"chamber_id"`
	Kind             string     `json:"kind"` // miter / lift / sector
	ClearWidthM      float64    `json:"clear_width_m"`
	Status           GateStatus `json:"status"`
	MisalignLimitDeg float64    `json:"misalign_limit_deg"`
	Enabled          bool       `json:"enabled"`
	CreatedAt        time.Time  `json:"created_at"`
}

// Validate 档案合法性：净宽与错位上限必须为正。
func (g *Gate) Validate() error {
	if g.ClearWidthM <= 0 {
		return errors.New("clear width must be positive")
	}
	if g.MisalignLimitDeg <= 0 {
		return errors.New("misalignment limit must be positive")
	}
	if g.ChamberID <= 0 {
		return errors.New("gate must belong to a chamber")
	}
	return nil
}

// Vessel 船舶档案。
type Vessel struct {
	ID      int64   `json:"id"`
	MMSI    string  `json:"mmsi"`
	Name    string  `json:"name"`
	LoraM   float64 `json:"lora_m"`
	BeamM   float64 `json:"beam_m"`
	DraftM  float64 `json:"draft_m"`
	Tonnage float64 `json:"tonnage"`
	Hazmat  bool    `json:"hazmat"`
}

// FitsChamber 船舶尺寸是否满足闸室通航净空。
func (v *Vessel) FitsChamber(c *Chamber) bool {
	return v.LoraM <= c.LengthM && v.BeamM <= c.WidthM
}

// Transit 一次过闸记录：进闸到出闸的全过程。
type Transit struct {
	ID         int64      `json:"id"`
	VesselID   int64      `json:"vessel_id"`
	ChamberID  int64      `json:"chamber_id"`
	Direction  string     `json:"direction"` // up / down
	EnteredAt  time.Time  `json:"entered_at"`
	ExitedAt   *time.Time `json:"exited_at"`
	WaitingSec int64      `json:"waiting_sec"`
	Priority   int        `json:"priority"`
}

// TransitDirection 合法方向集合。
var TransitDirection = map[string]bool{"up": true, "down": true}
