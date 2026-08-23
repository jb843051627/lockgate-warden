package service

import (
	"fmt"
	"sort"
	"time"

	"github.com/jb843051627/lockgate-warden/internal/model"
)

// Pilot 引航员档案。
type Pilot struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	License   string    `json:"license"` // A=危险品 B=客船 C=普货
	HazmatOK  bool      `json:"hazmat_ok"`
	OnDuty    bool      `json:"on_duty"`
	ShiftEnd  time.Time `json:"shift_end"`
	AssignCnt int64     `json:"assign_cnt"`
}

var pilotSeq int64

// pilots 进程内排班表（演示规模，落盘由 ops_events 审计兜底）。
var pilots = map[int64]*Pilot{}

// AddPilot 登记引航员。
func (s *Service) AddPilot(p *Pilot) error {
	if p.Name == "" || p.License == "" {
		return fmt.Errorf("pilot name and license are required")
	}
	if p.License != "A" && p.License != "B" && p.License != "C" {
		return fmt.Errorf("license must be A/B/C")
	}
	pilotSeq++
	p.ID = pilotSeq
	pilots[p.ID] = p
	return nil
}

// SetDuty 上下岗切换；下班前必须无未完成指派（简化为计数清零语义）。
func (s *Service) SetDuty(pilotID int64, onDuty bool, shiftEnd time.Time) error {
	p, ok := pilots[pilotID]
	if !ok {
		return fmt.Errorf("pilot %d: %w", pilotID, model.ErrNotFound)
	}
	if !onDuty && p.AssignCnt > 0 {
		return fmt.Errorf("%w: pilot %s has %d open assignments", model.ErrConflict, p.Name, p.AssignCnt)
	}
	p.OnDuty = onDuty
	if onDuty {
		p.ShiftEnd = shiftEnd
	}
	return nil
}

// AssignPilot 为过闸指派引航员：危险品必须 A 照，优先分配任务最少者。
func (s *Service) AssignPilot(vesselID int64, hazmat bool) (*Pilot, error) {
	v, err := s.store.GetVessel(vesselID)
	if err != nil {
		return nil, fmt.Errorf("vessel %d: %w", vesselID, err)
	}
	now := s.clock.Now()
	var best *Pilot
	for _, p := range pilots {
		if !p.OnDuty || p.ShiftEnd.Before(now.Add(30*time.Minute)) {
			continue
		}
		if (hazmat || v.Hazmat) && !p.HazmatOK {
			continue
		}
		if best == nil || p.AssignCnt < best.AssignCnt {
			best = p
		}
	}
	if best == nil {
		return nil, fmt.Errorf("%w: no qualified pilot on duty", model.ErrNotFound)
	}
	best.AssignCnt++
	return best, nil
}

// ReleasePilot 归还引航员任务额度。
func (s *Service) ReleasePilot(pilotID int64) error {
	p, ok := pilots[pilotID]
	if !ok {
		return fmt.Errorf("pilot %d: %w", pilotID, model.ErrNotFound)
	}
	if p.AssignCnt > 0 {
		p.AssignCnt--
	}
	return nil
}

// OnDutyRoster 当值名单：按任务数升序。
func (s *Service) OnDutyRoster() []Pilot {
	out := make([]Pilot, 0, len(pilots))
	for _, p := range pilots {
		if p.OnDuty {
			out = append(out, *p)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].AssignCnt < out[j].AssignCnt })
	return out
}

// OnDutyRosterSafe API 视图别名。
func (s *Service) OnDutyRosterSafe() ([]Pilot, error) {
	return s.OnDutyRoster(), nil
}

// AddPilotEntry API 入口：字段清洗后登记。
func (s *Service) AddPilotEntry(name, license string, hazmatOK bool) (*Pilot, error) {
	p := &Pilot{Name: name, License: license, HazmatOK: hazmatOK}
	if err := s.AddPilot(p); err != nil {
		return nil, err
	}
	return p, nil
}
